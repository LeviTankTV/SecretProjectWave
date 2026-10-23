package main

import (
	"math/rand"
	"sort"
	"strconv"
)

// statelessProfile — состояние профиля игрока, независимое от мира.
// Используется для правок инвентаря/хотбаров В ГЛАВНОМ МЕНЮ через REST,
// когда игрок НЕ находится в мире (нет живого Player-объекта).
type statelessProfile struct {
	inv       []InvStack
	hotbars   [2][]PetalDef
	activeBar int
	nick      string
	level     int
	pending   []PetalDef // результаты удачного крафта, ждут «Забрать»
	countG    int        // генератор ID, только для меню (не сохраняется)
}

func (sp *statelessProfile) clampBarSlot(bar, slot int) (int, int, bool) {
	if bar < 0 || bar > 1 {
		return 0, 0, false
	}
	// Кап = по текущему уровню игрока (как в мире). Слот ЗА капом — отклоняем,
	// а не молча срезаем на cap-1 (иначе «6-й слот» превратится в 5-й).
	cap := petalCapForLevel(sp.level)
	if slot < 0 || slot >= cap {
		return bar, slot, false
	}
	return bar, slot, true
}

func (sp *statelessProfile) hotbarAt(bar, slot int) (*PetalDef, bool) {
	if bar < 0 || bar > 1 {
		return nil, false
	}
	if slot < 0 || slot >= len(sp.hotbars[bar]) {
		return nil, false
	}
	d := sp.hotbars[bar][slot]
	if d.ID == "" {
		return nil, false
	}
	cp := d
	return &cp, true
}

func (sp *statelessProfile) setHotbarAt(bar, slot int, def *PetalDef) {
	if bar < 0 || bar > 1 || slot < 0 {
		return
	}
	hb := sp.hotbars[bar]
	if slot >= len(hb) {
		grow := make([]PetalDef, slot-len(hb)+1)
		hb = append(hb, grow...)
	}
	if def == nil {
		hb[slot] = PetalDef{}
	} else {
		hb[slot] = *def
	}
	sp.hotbars[bar] = hb
}

// findStack — стак вида/редкости с САМЫМ ВЫСОКИМ потенциалом (порт родителя):
// клик/надеть/выбросить берёт лучший стак. Катушка крафта — наоборот,
// findStackLowest: годролл не должен сгорать «автоматически».
func (sp *statelessProfile) findStack(kind string, rarity int) int {
	best := -1
	for i := range sp.inv {
		if sp.inv[i].Kind != kind || sp.inv[i].Rarity != rarity {
			continue
		}
		if sp.inv[i].UID != "" {
			continue // уникальный — только по UID
		}
		if best < 0 || normPot(sp.inv[i].Pot) > normPot(sp.inv[best].Pot) {
			best = i
		}
	}
	return best
}

// findStackLowest — стак того же вида/редкости с наименьшим потенциалом
// (тратится катушкой крафта).
func (sp *statelessProfile) findStackLowest(kind string, rarity int) int {
	best := -1
	for i := range sp.inv {
		if sp.inv[i].Kind != kind || sp.inv[i].Rarity != rarity {
			continue
		}
		if sp.inv[i].UID != "" {
			continue
		}
		if best < 0 || normPot(sp.inv[i].Pot) < normPot(sp.inv[best].Pot) {
			best = i
		}
	}
	return best
}

// findStackExact — точный стак по (вид, редкость, грейд).
func (sp *statelessProfile) findStackExact(kind string, rarity, pot int) int {
	for i := range sp.inv {
		if sp.inv[i].Kind == kind && sp.inv[i].Rarity == rarity &&
			normPot(sp.inv[i].Pot) == normPot(pot) {
			return i
		}
	}
	return -1
}

func (sp *statelessProfile) addToInventory(def PetalDef) {
	if i := sp.findStackExact(def.Kind, def.Rarity, def.Pot); i >= 0 {
		sp.inv[i].Count++
		sp.sort()
		return
	}
	s := InvStack{PetalDef: def, Count: 1}
	sp.inv = append(sp.inv, s)
	sp.sort()
}

func (sp *statelessProfile) takeOneFromInventory(kind string, rarity int) {
	i := sp.findStack(kind, rarity)
	if i < 0 {
		return
	}
	if sp.inv[i].Count > 1 {
		sp.inv[i].Count--
		return
	}
	sp.inv = append(sp.inv[:i], sp.inv[i+1:]...)
}

// takeN отнимает n штук из стака (по виду/редкости), удаляя стак если опустел.
func (sp *statelessProfile) takeN(kind string, rarity, n int) {
	i := sp.findStack(kind, rarity)
	if i < 0 {
		return
	}
	if sp.inv[i].Count > n {
		sp.inv[i].Count -= n
		return
	}
	sp.inv = append(sp.inv[:i], sp.inv[i+1:]...)
}

func (sp *statelessProfile) sort() {
	sort.SliceStable(sp.inv, func(a, b int) bool {
		if sp.inv[a].Rarity != sp.inv[b].Rarity {
			return sp.inv[a].Rarity > sp.inv[b].Rarity
		}
		return normPot(sp.inv[a].Pot) > normPot(sp.inv[b].Pot)
	})
}

func (sp *statelessProfile) freshID() string {
	sp.countG++
	return "hv_menu_" + sp.nick + "_" + strconv.Itoa(rand.Int()) + "_" + strconv.Itoa(sp.countG)
}

// move выполняет перемещение петаля между инвентарём и хотбарами (аналог
// Room.MovePetal, но на данных профиля). Возвращает true, если что-то изменилось.
func (sp *statelessProfile) move(from map[string]interface{}, toBar, toSlot int) bool {
	toBar, toSlot, ok := sp.clampBarSlot(toBar, toSlot)
	if !ok {
		return false
	}

	var carried *PetalDef
	fromHot := false
	var fromBar, fromSlot int

	src := "inv"
	if v, has := from["src"]; has {
		if s, ok2 := v.(string); ok2 {
			src = s
		}
	}
	if src == "hot" {
		fb, _ := from["bar"].(float64)
		fs, _ := from["slot"].(float64)
		fromBar = int(fb)
		fromSlot = int(fs)
		if def, ok2 := sp.hotbarAt(fromBar, fromSlot); ok2 {
			carried = def
			fromHot = true
		}
	} else {
		kind, _ := from["kind"].(string)
		rarity := int(from["rarity"].(float64))
		i := sp.findStack(kind, rarity)
		if i < 0 || sp.inv[i].Count <= 0 {
			return false
		}
		d := sp.inv[i].PetalDef
		carried = &d
	}

	if carried == nil {
		return false
	}

	targetDef, targetOccupied := sp.hotbarAt(toBar, toSlot)

	if fromHot && fromBar == toBar && fromSlot == toSlot {
		return false
	}

	if fromHot {
		sp.setHotbarAt(fromBar, fromSlot, nil)
	}
	if targetOccupied && targetDef.ID != "" {
		if fromHot {
			sp.setHotbarAt(fromBar, fromSlot, targetDef)
		} else {
			sp.addToInventory(*targetDef)
		}
	}

	eq := *carried
	eq.ID = sp.freshID()
	sp.setHotbarAt(toBar, toSlot, &eq)

	if !fromHot {
		kind, _ := from["kind"].(string)
		rarity := int(from["rarity"].(float64))
		sp.takeOneFromInventory(kind, rarity)
	}
	return true
}

// unequip возвращает предмет со слота хотбара обратно в инвентарь.
func (sp *statelessProfile) unequip(bar, slot int) bool {
	bar, slot, ok := sp.clampBarSlot(bar, slot)
	if !ok {
		return false
	}
	def, present := sp.hotbarAt(bar, slot)
	if !present {
		return false
	}
	sp.setHotbarAt(bar, slot, nil)
	sp.addToInventory(*def)
	return true
}

// drop уничтожает предмет (из инвентаря или хотбара) без XP — в меню наград нет.
func (sp *statelessProfile) drop(from map[string]interface{}) bool {
	src := "inv"
	if v, has := from["src"]; has {
		if s, ok2 := v.(string); ok2 {
			src = s
		}
	}
	if src == "hot" {
		bar := int(from["bar"].(float64))
		slot := int(from["slot"].(float64))
		bar, slot, ok := sp.clampBarSlot(bar, slot)
		if !ok {
			return false
		}
		if _, present := sp.hotbarAt(bar, slot); !present {
			return false
		}
		sp.setHotbarAt(bar, slot, nil)
		return true
	}
	kind, _ := from["kind"].(string)
	rarity := int(from["rarity"].(float64))
	i := sp.findStack(kind, rarity)
	if i < 0 || sp.inv[i].Count <= 0 {
		return false
	}
	sp.takeOneFromInventory(kind, rarity)
	return true
}

// setActiveBar переключает активную линейку (только 0 или 1).
func (sp *statelessProfile) setActiveBar(bar int) bool {
	if bar == 0 || bar == 1 {
		sp.activeBar = bar
		return true
	}
	return false
}

// addCountToStack добавляет n петалей заданного вида/редкости в запас
// (мержится с уже существующим стаком; если стака нет — создаёт его).
func (sp *statelessProfile) addCountToStack(kind string, rarity int, n int) {
	if n <= 0 {
		return
	}
	if i := sp.findStack(kind, rarity); i >= 0 {
		sp.inv[i].Count += n
		return
	}
	kn := petalKinds[kind]
	dmg := kn.BaseDMG * petalPower[rarity]
	hp := kn.BaseHP * petalPower[rarity]
	def := PetalDef{
		ID:     "craft_ref_" + strconv.Itoa(rand.Int()),
		Kind:   kind,
		Rarity: rarity,
		HP:     hp,
		MaxHP:  hp,
		DMG:    dmg,
		Size:   petalSizeBase,
		Color:  groundPetalColor(rarity),
		Name:   kn.Name,
	}
	// Фиксированные статы: размер (и всё остальное) — строго от вида+редкости,
	// идентично дропу (иначе крафтовые петали отличались размером от дроповых).
	normalizePetalDef(&def)
	sp.inv = append(sp.inv, InvStack{PetalDef: def, Count: n})
	sp.sort()
}

// craft крутит катушку (kind, rarity) — вложено invest штук (>=5).
// КАТУШКА КРУТИТ ВСЁ: попытки по 5 идут одна за другой, пока в пуле вложенных
// остаётся 5+. Провал теряет 1..4, ОСТАЛЬНОЕ (5-l) ВОЗВРАЩАЕТСЯ В ПУЛЬ И
// ДОКРУЧИВАЕТСЯ СЛЕДУЮЩЕЙ ПОПЫТКОЙ. В конце пула остаётся 1..4 петалей —
// они возвращаются в инвентарь. Итог: сколько вложил — в инвентаре остаётся
// 1..4 (плюс результаты и необращённый излишек стака сверх invest).
// Возвращает (status, successes, lost, results, batches): batches — лог каждого
// запуска (нужен клиенту для анимации).
func (sp *statelessProfile) craft(kind string, rarity int, invest int) (string, int, int, []PetalDef, []CraftBatch) {
	if !craftableRarity(rarity) {
		return "error", 0, 0, nil, nil
	}
	i := sp.findStackLowest(kind, rarity)
	if i < 0 || sp.inv[i].Count < invest {
		return "error", 0, 0, nil, nil
	}
	if invest < 5 {
		return "error", 0, 0, nil, nil
	}
	pct := craftChancePct[rarity]
	successes := 0
	lost := 0
	results := make([]PetalDef, 0, invest/5)
	batches := make([]CraftBatch, 0, invest/5)
	drawn := 0     // всего пропущено через катушку (для статистики)
	pool := invest // пул вложенных: тает на 5 за успех и на 1..4 за провал
	for pool >= 5 {
		// Достаём стопку заново (страховка: стак мог измениться).
		sidx := sp.findStackLowest(kind, rarity)
		if sidx < 0 || sp.inv[sidx].Count < 5 {
			break
		}
		sp.inv[sidx].Count -= 5
		pool -= 5
		drawn += 5
		roll := rand.Float64() * 100
		if roll <= pct {
			nd := makeCraftPetalBy(kind, rarity, sp.nick)
			results = append(results, nd)
			successes++
			batches = append(batches, []CraftBatch{CraftBatch{Success: true, Lost: 0}}...)
		} else {
			l := 1 + rand.Intn(4) // 1..4
			lost += l
			// Возврат 5-l: в стопку И в пул — они будут докручены.
			sp.addCountToStack(kind, rarity, 5-l) // возврат в тот же (нижний) стак
			pool += 5 - l
			batches = append(batches, []CraftBatch{CraftBatch{Success: false, Lost: l}}...)
		}
		// Защита от бесконечного цикла (не должно случиться: пул тает на 1..5 за попытку).
		if len(batches) > 100000 {
			break
		}
	}
	sp.pending = append(sp.pending, results...)
	if successes > 0 {
		return "success", successes, lost, results, batches
	}
	return "fail", 0, lost, results, batches
}

// takeCraft передаёт результаты неудачных... нет: все успешные крафты из "окна"
// в инвентарь. Возвращает true, если был отложенный результат.
func (sp *statelessProfile) takeCraft() bool {
	if sp.pending == nil {
		return false
	}
	for _, def := range sp.pending {
		sp.addToInventory(def)
	}
	sp.pending = nil
	return true
}
