package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

// Секретный ключ для подписи токенов. Хранится в серверном коде;
// токены подписываются им и потому переживают рестарт сервера (в отличие
// от старой схемы с счётчиком в памяти «sessions»).
var tokenSecret = []byte("secretproject-super-secret-key-change-me")

// Допустимое время жизни токена.
const tokenTTL = 30 * 24 * time.Hour

type User struct {
	Nick         string  `bson:"nick"`
	PasswordHash string  `bson:"password_hash"`
	Level        int     `bson:"level"`
	XP           float64 `bson:"xp"`
	// Раскладка и запас петалей храним как JSON-строки (плоско и безопасно для бсон).
	Inventory      string `bson:"inventory_json"` // JSON []InvStack
	Hotbars        string `bson:"hotbars_json"`   // JSON [2][]PetalDef
	ActiveBar      int    `bson:"active_bar"`
	CraftPending   string `bson:"craft_pending_json"`        // JSON PetalDef (результат крафта в ожидании) или ""
	Banned         bool   `bson:"banned"`                    // забанен ли аккаунт (админ-бан)
	CheckpointZone int    `bson:"checkpoint_zone,omitempty"` // зона чекпоинта (спавнпоинта)
	BestWave       int    `bson:"best_wave,omitempty"`       // рекорд: лучшая достигнутая волна

	// Миграция прогрессии (порт родителя): источник истины — накопленный опыт
	// XPTotal, (Level, XP) — производные кривой xpForLevel. prog_ver < 2 —
	// старая линейная кривая 30+12×L, конвертируется при первой загрузке.
	XPTotal float64 `bson:"xp_total,omitempty"`
	ProgVer int     `bson:"prog_ver,omitempty"`
	// Волновые достижения (2.0): айди заработанных милстоунов волны.
	Achievements []string `bson:"achievements,omitempty"`
}

var usersCollection *mongo.Collection

func initDB() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 127.0.0.1 вместо "localhost": go-драйвер иногда встаёт на IPv6-резолве
	// локального имени и не успевает переключиться на IPv4 до таймаута пула.
	mongoURI := "mongodb://127.0.0.1:27017"
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal("MongoDB connect error:", err)
	}
	if err = client.Ping(ctx, nil); err != nil {
		log.Fatal("MongoDB ping error:", err)
	}

	db := client.Database("secretprojectwave")
	usersCollection = db.Collection("users")

	// Unique index on nick.
	idx := mongo.IndexModel{
		Keys:    bson.D{{Key: "nick", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	if _, err := usersCollection.Indexes().CreateOne(ctx, idx); err != nil {
		log.Fatal("MongoDB index error:", err)
	}

	log.Println("MongoDB подключена, коллекция users готова")
}

func registerUser(nick, password string) (string, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	count, err := usersCollection.CountDocuments(ctx, bson.M{"nick": nick})
	if err != nil {
		return "", "Database error"
	}
	if count > 0 {
		return "", "Nickname is already taken"
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", "Hashing error"
	}

	// Стартовый набор: 5 Oval (Point) сразу в первом хотбаре, чтобы в меню
	// новчика сразу было видно снаряжение (как в игре).
	starter := make([]PetalDef, 0, 5)
	for i := 0; i < 5; i++ {
		starter = append(starter, basePetalDef("new_"+nick, i))
	}
	hotJSON, _ := json.Marshal([2][]PetalDef{starter, nil})
	if hotJSON == nil {
		hotJSON = []byte("[[],[]]")
	}

	_, err = usersCollection.InsertOne(ctx, User{
		Nick:         nick,
		PasswordHash: string(hash),
		Hotbars:      string(hotJSON),
	})
	if err != nil {
		return "", "Failed to create account"
	}

	token, _ := generateToken(nick)
	return token, ""
}

func loginUser(nick, password string) (string, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var user User
	err := usersCollection.FindOne(ctx, bson.M{"nick": nick}).Decode(&user)
	if err != nil {
		return "", "Invalid nickname or password"
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", "Invalid nickname or password"
	}
	if user.Banned {
		return "", "This account is banned"
	}

	token, _ := generateToken(nick)
	return token, ""
}

// generateToken создаёт подписанный токен: base64url(nick).base64url(exp).sig
// Сигнатура гарантирует неизменность и принадлежность нику; токен валиден
// даже после рестарта сервера (не хранится в памяти).
func generateToken(nick string) (string, string) {
	exp := time.Now().Add(tokenTTL).Unix()
	payload := struct {
		Nick string `json:"nick"`
		Exp  int64  `json:"exp"`
	}{nick, exp}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", "Token error"
	}
	part := base64.RawURLEncoding.EncodeToString(raw)
	sig := signToken(part)
	return part + "." + sig, ""
}

func signToken(part string) string {
	mac := hmac.New(sha256.New, tokenSecret)
	mac.Write([]byte(part))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// validateToken проверяет подпись и срок действия токена, возвращает ник.
func validateToken(token string) (string, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return "", false
	}
	part, sig := parts[0], parts[1]

	// Проверяем подпись.
	expected := signToken(part)
	if !hmac.Equal([]byte(expected), []byte(sig)) {
		return "", false
	}

	// Декодируем payload.
	raw, err := base64.RawURLEncoding.DecodeString(part)
	if err != nil {
		return "", false
	}
	var payload struct {
		Nick string `json:"nick"`
		Exp  int64  `json:"exp"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", false
	}

	// Проверяем срок действия.
	if payload.Exp < time.Now().Unix() {
		return "", false
	}
	return payload.Nick, true
}

// removeSession больше не требуется (токены самодостаточны и не хранятся
// в памяти), оставлена как совместимая заглушка.
func removeSession(token string) {}

// userExists — есть ли аккаунт с таким ником в БД. Живой (подписанный) токен
// действителен даже после удаления аккаунта из БД: без этой проверки игрок
// мог бы играть «призраком» — прогресс и петали молча НЕ сохранялись
// (UpdateOne по несуществующему нику ничего не делает).
func userExists(nick string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return usersCollection.FindOne(ctx, bson.M{"nick": nick}).Err() == nil
}

// loadProgress возвращает сохранённый уровень и XP игрока (по умолчанию 0).
func loadProgress(nick string) (int, float64) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var user User
	err := usersCollection.FindOne(ctx, bson.M{"nick": nick}).Decode(&user)
	if err != nil {
		return 0, 0
	}
	return user.Level, user.XP
}

// saveProgress сохраняет уровень и XP игрока в БД.
// Юзер уже существует (зарегистрирован), обновляем ТОЛЬКО level и xp,
// не трогая пароль. Upsert НЕ используется, чтобы не затереть пароль.
func saveProgress(nick string, level int, xp float64) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	usersCollection.UpdateOne(ctx, bson.M{"nick": nick}, bson.M{
		"$set": bson.M{"level": level, "xp": xp},
	})
}

// loadProfile возвращает полный прогресс игрока: уровень, XP, запас петалей
// (инвентарь-стаки), раскладку хотбаров и активную линейку.
// Для новых игроков возвращает нули/пустые значения.
func loadProfile(nick string) (level int, xp float64, inv []InvStack, hotbars [2][]PetalDef, activeBar int) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var u User
	err := usersCollection.FindOne(ctx, bson.M{"nick": nick}).Decode(&u)
	if err != nil {
		return 0, 0, nil, [2][]PetalDef{}, 0
	}
	if u.Inventory != "" {
		_ = json.Unmarshal([]byte(u.Inventory), &inv)
	}
	if u.Hotbars != "" {
		_ = json.Unmarshal([]byte(u.Hotbars), &hotbars)
	}
	// Нормализация: каждый петаль приводится к ФИКСИРОВАННЫМ статам своей пары
	// (вид, редкость) — старые сохранения БД с любыми HP/DMG/размером не влияют.
	for i := range inv {
		normalizePetalDef(&inv[i].PetalDef)
	}
	for b := range hotbars {
		for i := range hotbars[b] {
			normalizePetalDef(&hotbars[b][i])
		}
	}
	if u.ActiveBar < 0 || u.ActiveBar > 1 {
		u.ActiveBar = 0
	}
	// Миграция прогрессии v2 (порт родителя): старая пара (level, xp) с линейной
	// кривой пересчитывается в накопленный XPTotal; уровень сохраняется по
	// построению (totalXPForLevel(level)+xp -> тот же level на новой кривой).
	// Результат сразу персистится, чтобы повторная загрузка не мигрировала дважды.
	if u.ProgVer < 2 {
		if u.XPTotal <= 0 {
			u.XPTotal = totalXPForLevel(u.Level) + u.XP
		}
		u.ProgVer = 2
		mctx, mcancel := context.WithTimeout(context.Background(), 2*time.Second)
		usersCollection.UpdateOne(mctx, bson.M{"nick": nick}, bson.M{
			"$set": bson.M{"xp_total": u.XPTotal, "prog_ver": u.ProgVer},
		})
		mcancel()
	}
	u.Level, u.XP = levelForTotalXP(u.XPTotal)
	return u.Level, u.XP, inv, hotbars, u.ActiveBar
}

// saveProfile сохраняет полный прогресс игрока в БД (уровень, XP, запас,
// хотбары и активную линейку). Пароль не трогаем.
func saveProfile(nick string, level int, xp float64, inv []InvStack, hotbars [2][]PetalDef, activeBar int) {
	invJSON, _ := json.Marshal(inv)
	if invJSON == nil {
		invJSON = []byte("[]")
	}
	hotJSON, _ := json.Marshal(hotbars)
	if hotJSON == nil {
		hotJSON = []byte("[[],[]]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	usersCollection.UpdateOne(ctx, bson.M{"nick": nick}, bson.M{
		"$set": bson.M{
			"level":          level,
			"xp":             xp,
			"inventory_json": string(invJSON),
			"hotbars_json":   string(hotJSON),
			"active_bar":     activeBar,
			"xp_total":       totalXPForLevel(level) + xp,
			"prog_ver":       2,
		},
	})
}

// loadBestWave — рекорд игрока «лучшая достигнутая волна».
func loadBestWave(nick string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var u User
	if err := usersCollection.FindOne(ctx, bson.M{"nick": nick}).Decode(&u); err != nil {
		return 0
	}
	return u.BestWave
}

// bumpBestWave поднимает рекорд волны, если w больше текущего.
func bumpBestWave(nick string, w int) {
	if nick == "" || w <= 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	usersCollection.UpdateOne(ctx,
		bson.M{"nick": nick, "best_wave": bson.M{"$lt": w}},
		bson.M{"$set": bson.M{"best_wave": w}})
	// Волновые достижения (2.0): милстоуны волны 10/25/50/100 добавляются
	// идемпотентно ($addToSet) при обновлении рекорда.
	for _, id := range waveAchievementsFor(w) {
		actx, acancel := context.WithTimeout(context.Background(), 2*time.Second)
		usersCollection.UpdateOne(actx,
			bson.M{"nick": nick},
			bson.M{"$addToSet": bson.M{"achievements": id}})
		acancel()
	}
}

// waveAchievementsFor — айди всех милстоунов, заработанных достижением волны w.
func waveAchievementsFor(w int) []string {
	var out []string
	for _, m := range []int{10, 25, 50, 100} {
		if w >= m {
			out = append(out, "wave"+strconv.Itoa(m))
		}
	}
	return out
}

// loadAchievements — список достижений игрока.
func loadAchievements(nick string) []string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var u User
	if err := usersCollection.FindOne(ctx, bson.M{"nick": nick}).Decode(&u); err != nil {
		return nil
	}
	return u.Achievements
}

// loadCheckpoint возвращает сохранённую зону чекпоинта игрока (0 — Point).
// Чекпоинт существует только для первого уровня (0-7): сохранённый второй
// уровень трактуется как 0, чтобы из L2 смерть/релог всегда вели в L1.
func loadCheckpoint(nick string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var u User
	if err := usersCollection.FindOne(ctx, bson.M{"nick": nick}).Decode(&u); err != nil {
		return 0
	}
	if u.CheckpointZone < 0 || u.CheckpointZone >= secondTierOffset {
		return 0
	}
	return u.CheckpointZone
}

// saveCheckpoint сохраняет зону чекпоинта игрока (вызывается асинхронно,
// только при СМЕНЕ чекпоинта — не каждый тик).
func saveCheckpoint(nick string, zone int) {
	if nick == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := usersCollection.UpdateOne(ctx, bson.M{"nick": nick},
		bson.M{"$set": bson.M{"checkpoint_zone": zone}})
	if err != nil {
		log.Println("checkpoint save error:", err)
	}
}

// loadPendingCraft возвращает отложенные результаты крафта игрока (или пустой).
func loadPendingCraft(nick string) []PetalDef {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var u User
	err := usersCollection.FindOne(ctx, bson.M{"nick": nick}).Decode(&u)
	if err != nil || u.CraftPending == "" {
		return nil
	}
	var defs []PetalDef
	if json.Unmarshal([]byte(u.CraftPending), &defs) != nil {
		return nil
	}
	return defs
}

// savePendingCraft сохраняет (nil — очищает) отложенные результаты крафта.
func savePendingCraft(nick string, defs []PetalDef) {
	val := ""
	if defs != nil {
		enc, _ := json.Marshal(defs)
		if enc != nil {
			val = string(enc)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	usersCollection.UpdateOne(ctx, bson.M{"nick": nick}, bson.M{
		"$set": bson.M{"craft_pending_json": val},
	})
}

// profileSnapshot — снимок состояния игрока для группового сохранения.
type profileSnapshot struct {
	Name    string
	Level   int
	XP      float64
	Inv     []InvStack
	Hotbars [2][]PetalDef
	Active  int
	Pending []PetalDef
}

// bulkSaveProfiles сохраняет снимки нескольких игроков ОДНИМ bulkWrite вместо
// N×2 отдельных запросов (кардинально уменьшает нагрузку на MongoDB при
// автосейве всех онлайн-игроков). Обновляются ТОЛЬКО поля прогресса через
// $set — password_hash/banned/checkpoint_zone не затрагиваются. Сериализация
// JSON выполняется здесь же (в отдельной фоновой горутине, не в игровом тике).
func bulkSaveProfiles(snaps []profileSnapshot) {
	if len(snaps) == 0 {
		return
	}
	models := make([]mongo.WriteModel, 0, len(snaps))
	for i := range snaps {
		s := &snaps[i]
		invJSON, _ := json.Marshal(s.Inv)
		if invJSON == nil {
			invJSON = []byte("[]")
		}
		hotJSON, _ := json.Marshal(s.Hotbars)
		if hotJSON == nil {
			hotJSON = []byte("[[],[]]")
		}
		pendVal := ""
		if s.Pending != nil {
			enc, _ := json.Marshal(s.Pending)
			if enc != nil {
				pendVal = string(enc)
			}
		}
		models = append(models, mongo.NewUpdateOneModel().
			SetFilter(bson.M{"nick": s.Name}).
			SetUpdate(bson.M{"$set": bson.M{
				"level":              s.Level,
				"xp":                 s.XP,
				"inventory_json":     string(invJSON),
				"hotbars_json":       string(hotJSON),
				"active_bar":         s.Active,
				"craft_pending_json": pendVal,
			}}))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := usersCollection.BulkWrite(ctx, models); err != nil {
		log.Printf("[autosave] bulkWrite error: %v", err)
	}
}
