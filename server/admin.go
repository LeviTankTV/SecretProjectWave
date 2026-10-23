package main

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"math/rand"
	"os"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// admin.go — админ-инструменты: спавн мобов, список игроков, бан, смена
// скорости/здоровья себе. Админ — единственный аккаунт с ником adminNick.

// adminNick — ник игрока-администратора. Задаётся переменной окружения
// ADMIN_NICK, по умолчанию "Admin". Смени при необходимости.
var adminNick = func() string {
	if v := os.Getenv("ADMIN_NICK"); v != "" {
		return v
	}
	return "Admin"
}()

// isAdmin возвращает true, если ник — администратор.
func isAdmin(nick string) bool {
	return nick == adminNick
}

// roomAdmin возвращает true, если клиент подключён аккаунтом админа.
func roomAdmin(c *Client) bool {
	c.room.mu.Lock()
	defer c.room.mu.Unlock()
	p, ok := c.room.players[c.playerID]
	if !ok {
		return false
	}
	return isAdmin(p.State.Name)
}

// userBanned проверяет в БД, забанен ли аккаунт.
func userBanned(nick string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var u User
	err := usersCollection.FindOne(ctx, bson.M{"nick": nick}).Decode(&u)
	if err != nil {
		return false
	}
	return u.Banned
}

// setBanned записывает статус бана аккаунта в БД.
func setBanned(nick string, banned bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, _ = usersCollection.UpdateOne(ctx, bson.M{"nick": nick}, bson.M{
		"$set": bson.M{"banned": banned},
	})
}

// adminSend шлёт JSON-сообщение конкретному клиенту (админу).
func adminSend(c *Client, v interface{}) {
	msg, _ := json.Marshal(v)
	c.deliver(msg)
}

// playerInfoMsg — компактная сводка об игроке для админ-панели.
type playerInfoMsg struct {
	Type  string  `json:"type"`
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Zone  int     `json:"zone"`
	HP    float64 `json:"hp"`
	MaxHP float64 `json:"maxHp"`
	Level int     `json:"level"`
}

// adminPlayersList шлёт админу список всех онлайн-игроков.
func (r *Room) adminPlayersList(c *Client) {
	r.mu.Lock()
	list := make([]playerInfoMsg, 0, len(r.players))
	for id, p := range r.players {
		num, _ := strconv.Atoi(id)
		list = append(list, playerInfoMsg{
			Type: "adminPlayers", ID: num, Name: p.State.Name,
			X: p.State.X, Y: p.State.Y, Zone: p.zone,
			HP: p.State.HP, MaxHP: p.State.MaxHP, Level: p.State.Level,
		})
	}
	r.mu.Unlock()
	adminSend(c, map[string]interface{}{"type": "adminPlayers", "players": list})
}

// adminSpawnMob спавнит моба выбранного вида и редкости на позиции админа.
// mtype — «дополнительный тип» моба: nothing (обычный) или gold (золотой/Shiny).
func (r *Room) adminSpawnMob(c *Client, kind string, rarity int, mtype string) {
	r.mu.Lock()
	p, ok := r.players[c.playerID]
	if !ok {
		r.mu.Unlock()
		return
	}
	zone := p.zone
	rar := rarity
	if rar < 0 {
		rar = 0
	}
	if rar > len(rarityNames)-1 {
		rar = len(rarityNames) - 1
	}
	if _, exists := mobTypes[kind]; !exists {
		kind = "dot"
	}
	r.mobIDSeq++
	m := makeAdminMob(kind, rar, zone, p.State.X, p.State.Y, r.mobIDSeq, mtype)
	r.mobs = append(r.mobs, m)
	r.mu.Unlock()

	if txt := superSpawnText(zone, m); txt != "" {
		r.BroadcastSystem("super", txt)
	}
	adminSend(c, map[string]interface{}{"type": "adminAck", "text": "Spawned " + mobTypes[m.Type].Name + " " + rarityNames[m.Rarity]})
}

// makeAdminMob создаёт моба конкретного вида и редкости в точке (x, y) зоны.
func makeAdminMob(kind string, rar int, zone int, x, y float64, id int, mtype string) Mob {
	mt := mobTypes[kind]
	speed := mt.BaseSpeed * (0.8 + 0.4)

	// Прижимаем точку внутрь Mob-зоны, чтобы моб не торчал за границей
	// и не попал в стены / Safe Zone.
	m := Mob{
		ID:        "mob_" + strconv.Itoa(id),
		Type:      kind,
		Rarity:    rar,
		X:         x,
		Y:         y,
		Speed:     speed,
		Zone:      zone,
		Damage:    make(map[string]float64),
		state:     "patrol",
		moveTimer: 0,
		dirX:      1,
		dirY:      0,
	}
	// «gold»/«ring»/«star»/«crescent» — принудительный подвид моба;
	// «nothing» и прочее — обычный.
	if mtype != "" && mtype != "nothing" {
		m.Variant = mtype
	}
	// Спецспособности: comet/arrow/beam инициализация как у обычных мобов.
	initSpecMobPhase(&m, kind)
	// Единый путь расчёта статов с обычными мобами: HP (включая множители
	// боссов Fractal ×10 / Chaos ×32), DMG, радиус/массу, порог спавна улья и
	// орбитальные петали. Раньше админ-босс считал HP только базовой формулой
	// и был в разы слабее реального босса той же редкости.
	scaleMobForRarity(&m, rar)
	clampMobInZone(&m)
	return m
}

// adminBan банит игрока по нику: кикает с сервера и помечает аккаунт.
func (r *Room) adminBan(c *Client, target string) {
	if target == "" {
		return
	}
	if !isAdmin(target) && target != adminNick {
		setBanned(target, true)
	}
	// Отключаем цель, если она онлайн.
	var targetClient *Client
	r.mu.Lock()
	for id, p := range r.players {
		if p.State.Name == target {
			if cl, ok := r.clients[id]; ok {
				targetClient = cl
			}
		}
	}
	r.mu.Unlock()
	if targetClient != nil {
		go func(cl *Client) { _ = cl.conn.Close() }(targetClient)
	}
	adminSend(c, map[string]interface{}{"type": "adminAck", "text": target + " banned"})
	log.Printf("Admin %q banned %q", adminNick, target)
}

// adminUnban снимает бан с аккаунта по нику.
func (r *Room) adminUnban(c *Client, target string) {
	if target == "" {
		return
	}
	setBanned(target, false)
	adminSend(c, map[string]interface{}{"type": "adminAck", "text": "Unbanned " + target})
	log.Printf("Admin %q unbanned %q", adminNick, target)
}

// adminSetSpeed меняет скорость перемещения админа.
func (r *Room) adminSetSpeed(c *Client, v float64) {
	if v < 1 {
		v = 1
	}
	if v > 5000 {
		v = 5000
	}
	r.mu.Lock()
	if p, ok := r.players[c.playerID]; ok {
		p.speed = v
	}
	r.mu.Unlock()
	adminSend(c, map[string]interface{}{"type": "adminAck", "text": "Speed: " + strconv.FormatFloat(v, 'f', 0, 64)})
}

// adminSetHP меняет здоровье админа (и текущее, и максимум).
func (r *Room) adminSetHP(c *Client, v float64) {
	if v < 1 {
		v = 1
	}
	if v > 100000 {
		v = 100000
	}
	r.mu.Lock()
	if p, ok := r.players[c.playerID]; ok {
		p.State.MaxHP = v
		p.State.HP = v
	}
	r.mu.Unlock()
	adminSend(c, map[string]interface{}{"type": "adminAck", "text": "Health: " + strconv.FormatFloat(v, 'f', 0, 64)})
}

const adminSizeMin = 2.0
const adminSizeMax = 500.0

// adminSizeDelta увеличивает/уменьшает размер (радиус) админа. delta — приращение
// к текущему размеру; отрицательное значение уменьшает. Результат ограничен.
func (r *Room) adminSizeDelta(c *Client, delta float64) {
	r.mu.Lock()
	if p, ok := r.players[c.playerID]; ok {
		p.State.Size += delta
		if p.State.Size < adminSizeMin {
			p.State.Size = adminSizeMin
		}
		if p.State.Size > adminSizeMax {
			p.State.Size = adminSizeMax
		}
	}
	r.mu.Unlock()
	god, inv, pas, pvp, size, shot := r.adminSelfState(c)
	adminSendState(c, god, inv, pas, pvp, size, shot)
	adminSend(c, map[string]interface{}{"type": "adminAck", "text": "Size: " + strconv.FormatFloat(size, 'f', 1, 64)})
}

// adminTeleportTo телепортирует админа к игроку по нику (в его зону и позицию).
func (r *Room) adminTeleportTo(c *Client, target string) {
	r.mu.Lock()
	ap, ok := r.players[c.playerID]
	if !ok {
		r.mu.Unlock()
		return
	}
	for _, tp := range r.players {
		if tp.State.Name == target {
			ap.zone = tp.zone
			ap.State.Zone = tp.zone
			ap.State.X = tp.State.X
			ap.State.Y = tp.State.Y
			ap.transferT = 0
			r.mu.Unlock()
			adminSend(c, map[string]interface{}{"type": "adminAck", "text": "Teleported to " + target})
			return
		}
	}
	r.mu.Unlock()
	adminSend(c, map[string]interface{}{"type": "adminAck", "text": "Player \"" + target + "\" not found"})
}

// adminTeleportZone телепортирует админа в центр зоны по индексу.
// zones: 0=Point, 1=Line, 2=Polygon, 3=Fractal, 4=Chaos.
func (r *Room) adminTeleportZone(c *Client, zone int) {
	if zone < 0 || zone >= len(zones) {
		adminSend(c, map[string]interface{}{"type": "adminAck", "text": "Unknown zone"})
		return
	}
	r.mu.Lock()
	ap, ok := r.players[c.playerID]
	if !ok {
		r.mu.Unlock()
		return
	}
	// Снимаем ожидание перехода (если админ застрял в портале) и ставим в центр.
	ap.transferTo = -1
	ap.transferT = 0
	r.placeInZone(ap, zone)
	r.mu.Unlock()
	adminSend(c, map[string]interface{}{"type": "adminAck", "text": "Teleported to center of " + zones[zone].Name})
}

// adminTeleportSafe телепортирует админа в ЦЕНТР Safe Zone зоны первого уровня
// (0-7). У зон второго уровня (8-15) Safe Zone нет — такие зоны отклоняем.
// Позиция — точно по центру безопасной полосы (низ зоны), без случайности,
// чтобы админ не «упал» в стену/моб-зону.
func (r *Room) adminTeleportSafe(c *Client, zone int) {
	if zone < 0 || zone >= secondTierOffset {
		adminSend(c, map[string]interface{}{"type": "adminAck", "text": "Safe zone exists only for first-tier zones (0-7)"})
		return
	}
	r.mu.Lock()
	ap, ok := r.players[c.playerID]
	if !ok {
		r.mu.Unlock()
		return
	}
	z := zones[zone]
	ap.transferTo = -1
	ap.transferT = 0
	ap.zone = zone
	ap.State.Zone = zone
	ap.State.X = z.X + z.W/2
	ap.State.Y = z.Y + z.H - safeZoneH/2
	r.mu.Unlock()
	adminSend(c, map[string]interface{}{"type": "adminAck", "text": "Teleported to center of " + zones[zone].Name + " safe zone"})
}

// adminPullPlayer телепортирует игрока по нику К администратору (в зону и
// позицию админа). Цели даём короткий иммунитет, чтобы она не умерла мгновенно
// в моб-зоне, и снимаем ожидание перехода портала.
func (r *Room) adminPullPlayer(c *Client, target string) {
	if target == "" {
		return
	}
	r.mu.Lock()
	ap, ok := r.players[c.playerID]
	if !ok {
		r.mu.Unlock()
		return
	}
	for _, tp := range r.players {
		if tp.State.Name == target {
			if tp.State.ID == ap.State.ID {
				r.mu.Unlock()
				adminSend(c, map[string]interface{}{"type": "adminAck", "text": "That's you already"})
				return
			}
			tp.transferTo = -1
			tp.transferT = 0
			tp.zone = ap.zone
			tp.State.Zone = ap.zone
			tp.State.X = ap.State.X
			tp.State.Y = ap.State.Y
			tp.invulnT = respawnInvuln
			r.mu.Unlock()
			adminSend(c, map[string]interface{}{"type": "adminAck", "text": "Pulled \"" + target + "\" to you"})
			return
		}
	}
	r.mu.Unlock()
	adminSend(c, map[string]interface{}{"type": "adminAck", "text": "Player \"" + target + "\" not found"})
}

// adminSelfState возвращает состояние неуязвимости/невидимости/пассивности/ПВП/
// ваншот-режима и размер админа.
func (r *Room) adminSelfState(c *Client) (god bool, invisible bool, passive bool, team int, size float64, oneshot bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.players[c.playerID]; ok {
		god = p.GodMode
		invisible = p.Invisible
		passive = p.Passive
		team = p.Team
		size = p.State.Size
		oneshot = p.OneShot
	}
	return
}

// adminSendState отправляет админу текущее состояние (неуязвимость/невидимость/…).
func adminSendState(c *Client, god bool, invisible bool, passive bool, team int, size float64, oneshot bool) {
	adminSend(c, map[string]interface{}{
		"type": "adminState", "god": god, "invisible": invisible,
		"passive": passive, "team": team, "size": size, "oneshot": oneshot,
	})
}

// adminToggleGod переключает неуязвимость админа.
func (r *Room) adminToggleGod(c *Client) {
	var on bool
	r.mu.Lock()
	if p, ok := r.players[c.playerID]; ok {
		p.GodMode = !p.GodMode
		on = p.GodMode
	}
	r.mu.Unlock()
	_, inv, pas, team, size, shot := r.adminSelfState(c)
	adminSendState(c, on, inv, pas, team, size, shot)
	adminSend(c, map[string]interface{}{"type": "adminAck", "text": "Invulnerability: " + onOff(on)})
}

// adminToggleInvisible переключает невидимость админа для других игроков.
func (r *Room) adminToggleInvisible(c *Client) {
	var on bool
	r.mu.Lock()
	if p, ok := r.players[c.playerID]; ok {
		p.Invisible = !p.Invisible
		on = p.Invisible
	}
	r.mu.Unlock()
	god, _, pas, team, size, shot := r.adminSelfState(c)
	adminSendState(c, god, on, pas, team, size, shot)
	adminSend(c, map[string]interface{}{"type": "adminAck", "text": "Invisibility: " + onOff(on)})
}

// adminTogglePassive переключает пассивность: мобы не агрятся на админа и не
// получают от него урона (админа его петали оставляют мобов нетронутыми).
func (r *Room) adminTogglePassive(c *Client) {
	var on bool
	r.mu.Lock()
	if p, ok := r.players[c.playerID]; ok {
		p.Passive = !p.Passive
		on = p.Passive
	}
	r.mu.Unlock()
	god, inv, _, team, size, shot := r.adminSelfState(c)
	adminSendState(c, god, inv, on, team, size, shot)
	adminSend(c, map[string]interface{}{"type": "adminAck", "text": "Passivity: " + onOff(on)})
}

// adminTogglePVP переключает админа в команду 1 (враждебный): его петали вредят
// другим игрокам и их петалям, и он враждебен мобам. Повторный клик — назад в 0.
func (r *Room) adminTogglePVP(c *Client) {
	var on bool
	r.mu.Lock()
	if p, ok := r.players[c.playerID]; ok {
		if p.Team == 1 {
			p.Team = 0
		} else {
			p.Team = 1
		}
		on = p.Team == 1
	}
	r.mu.Unlock()
	god, inv, pas, team, size, shot := r.adminSelfState(c)
	adminSendState(c, god, inv, pas, team, size, shot)
	adminSend(c, map[string]interface{}{"type": "adminAck", "text": "Team 1 (hostile): " + onOff(on)})
}

// adminToggleMobTeam переключает админа в команду 2 (на стороне мобов): враждебен
// игрокам, но НЕ взаимодействует с мобами (не бьёт их, не получает урона, мобы не
// агрятся). Повторный клик — назад в 0.
func (r *Room) adminToggleMobTeam(c *Client) {
	var on bool
	r.mu.Lock()
	if p, ok := r.players[c.playerID]; ok {
		if p.Team == 2 {
			p.Team = 0
		} else {
			p.Team = 2
		}
		on = p.Team == 2
	}
	r.mu.Unlock()
	god, inv, pas, team, size, shot := r.adminSelfState(c)
	adminSendState(c, god, inv, pas, team, size, shot)
	adminSend(c, map[string]interface{}{"type": "adminAck", "text": "Team 2 (mob team): " + onOff(on)})
}

// adminToggleOneShot переключает ваншот-режим: касание телом мгновенно убивает
// любого моба зоны — удобно проверять убийство/смерть и дроп больших боссов.
func (r *Room) adminToggleOneShot(c *Client) {
	var on bool
	r.mu.Lock()
	if p, ok := r.players[c.playerID]; ok {
		p.OneShot = !p.OneShot
		on = p.OneShot
	}
	r.mu.Unlock()
	god, inv, pas, team, size, shot := r.adminSelfState(c)
	adminSendState(c, god, inv, pas, team, size, shot)
	adminSend(c, map[string]interface{}{"type": "adminAck", "text": "One-shot body: " + onOff(on)})
}

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

// petalDefForAdmin строит предмет петали по виду и редкости.
func petalDefForAdmin(kind string, rarity int) (PetalDef, bool) {
	if kind == "" {
		kind = "oval"
	}
	kn, ok := petalKinds[kind]
	if !ok {
		return PetalDef{}, false
	}
	mult := petalPower[rarity]
	size := petalSizeBase * math.Pow(1.2, float64(rarity))
	hp := kn.BaseHP * mult
	def := PetalDef{
		ID:   "ad_" + strconv.Itoa(rand.Int()),
		Kind: kind, Rarity: rarity,
		HP: hp, MaxHP: hp, DMG: kn.BaseDMG * mult,
		Size: size, Color: groundPetalColor(rarity), Name: kn.Name,
	}
	normalizePetalDef(&def)
	return def, true
}

// adminGivePetals даёт админу count петалей заданного вида и редкости в инвентарь.
func (r *Room) adminGivePetals(c *Client, kind string, rarity int, count int) {
	if rarity < 0 {
		rarity = 0
	}
	if rarity > len(rarityNames)-1 {
		rarity = len(rarityNames) - 1
	}
	if count < 1 {
		count = 1
	}
	if count > 999 {
		count = 999
	}
	def, ok := petalDefForAdmin(kind, rarity)
	if !ok {
		adminSend(c, map[string]interface{}{"type": "adminAck", "text": "Unknown petal type"})
		return
	}
	r.mu.Lock()
	if p, ok := r.players[c.playerID]; ok {
		for i := 0; i < count; i++ {
			r.addToInventory(p, def)
		}
	}
	r.mu.Unlock()
	adminSend(c, map[string]interface{}{"type": "adminAck", "text": "Gave " + strconv.Itoa(count) + "x " + def.Name + " " + zoneNames[def.Rarity]})
}

// adminGiveXP начисляет админу опыт (с прокачкой уровня).
func (r *Room) adminGiveXP(c *Client, amount float64) {
	if amount < 1 {
		amount = 1
	}
	if amount > 1000000 {
		amount = 1000000
	}
	r.mu.Lock()
	if p, ok := r.players[c.playerID]; ok {
		p.awardXP(amount)
		p.ensurePetalCap()
	}
	r.mu.Unlock()
	adminSend(c, map[string]interface{}{"type": "adminAck", "text": "XP +" + strconv.FormatFloat(amount, 'f', 0, 64)})
}

// adminAnnounce шлёт всем игрокам системное объявление.
func (r *Room) adminAnnounce(c *Client, text string) {
	if text == "" {
		text = "Announcement"
	}
	r.BroadcastSystem("announce", text)
	adminSend(c, map[string]interface{}{"type": "adminAck", "text": "Announcement sent"})
}

// zoneMobInfo — сведения о мобах в одной зоне для админ-panels.
type zoneMobInfo struct {
	Index int    `json:"index"`
	Name  string `json:"name"`
	Count int    `json:"count"`
	Limit int    `json:"limit"`
	// Hexagon: живые мини-боссы в зоне (кап 2). Fractal по таймеру больше
	// не существует (ролл 1/1500 в верхней полосе).
	HexHere int `json:"hexHere"`
}

// adminMobCounts шлёт админу число мобов по каждой зоне.
func (r *Room) adminMobCounts(c *Client) {
	r.mu.Lock()
	list := make([]zoneMobInfo, 0, len(zones))
	for i := range zones {
		fi := zoneMobInfo{
			Index: i, Name: zones[i].Name,
			Count: r.mobsCount(i), Limit: zoneMobLimits[i],
			HexHere: r.hexInZone(i),
		}
		list = append(list, fi)
	}
	r.mu.Unlock()
	adminSend(c, map[string]interface{}{"type": "adminMobs", "zones": list})
}
