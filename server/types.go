package main

// types.go - общие типы, разделяемые между файлами сервера.

type Vec struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Rect — ось-выровненный прямоугольник (для стен зон).
type Rect struct {
	X, Y, W, H float64
}

type PlayerState struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Size  float64 `json:"size"`
	Color uint32  `json:"color"`
	Zone  int     `json:"zone"`
	HP    float64 `json:"hp"`
	MaxHP float64 `json:"maxHp"`
	Shield   float64 `json:"shield"`   // щит-броня: поглощает урон раньше HP
	MaxShield float64 `json:"maxShield"`
	Team  int     `json:"team"`
	Dead  bool    `json:"dead"`
	Level int     `json:"level"`
	XP    float64 `json:"xp"`
	XPMax float64 `json:"xpMax"`

	Inventory []InvStack `json:"inventory"`    // стаки петалей в инвентаре (запас)
	Hotbars   [2][]PetalDef `json:"hotbars"`   // две линейки хотбара
	ActiveBar int          `json:"activeBar"`  // активная линейка (0/1)
	MaxHotRar int          `json:"maxHotRar"` // макс. редкость хотбара (цвет уровня) для других игроков
}

type Player struct {
	State PlayerState

	speed      float64
	input      Vec
	lastDir    Vec  // последнее направление движения (для рывка петеля void)
	velX, velY float64 // текущая скорость игрока (сглаживание движения)
	zone       int
	viewW      float64 // размер окна/камеры клиента (для фильтра видимости)
	viewH      float64
	portalCD   float64
	transferTo int
	transferT  float64
	Dead       bool
	checkpoint int // зона закреплённого чекпоинта (спавнпоинта)

	petals    []Petal
	inventory []InvStack // стаки петалей в инвентаре (запас)
	hotbars   [2][]PetalDef // две линейки хотбара (каждая до капа уровня)
	activeBar int           // активная линейка (0 или 1)
	// Внутриигровой крафт: отложенные результаты на игроке (в БД при выходе).
	pendingCraft []PetalDef
	petalDist float64
	petalDir  float64
	baseAngle float64

	mobHitCD float64
	invulnT  float64 // иммунитет к урону (сек) после респавна/портала

	GodMode   bool // неуязвимость (админ)
	Invisible bool // невидим для остальных игроков (админ)
	Passive   bool // пассивность: мобы не агрятся на админа и не получают от него урон
	Team      int  // команда: 0 = обычный (PvE, дружелюбен всем), 1 = враждебный (PvP-рого), 2 = команда мобов (только админ)
	OneShot   bool // тест-режим: тело админа убивает мобов касанием (для дропов больших мобов)

	// Компас (петаль compass): таймер до следующей отправки направления на цель.
	compassT float64

// Глаз (петаль eye): РАВНОМЕРНЫЙ ramp текущего множителя радиуса орбиты к
	// целевому (1 + eyeOrbitPerRar×редкость) — иначе радиус «прыгал» при
	// надевании/поломке глаза. 0 = ещё не инициализирован.
	eyeRamp float64

	// wallHold — накопитель «вдавливания» в стену (слайми-отталкивание):
	// растёт, пока игрок ЖМЁТ движение в стену (кламп реально удерживает),
	// спадает при отпускании/отходе. При wallHold >= slimeWallRampTime push
	// достигает максимума и стена выталкивает игрока (АФК у стены невозможен),
	// но обычный подход/касание стены остаются свободными.
	wallHold float64

	// noDamageT — секунды БЕЗ получения урона (реген вне боя): по достижении
	// outOfCombatDelay игрок медленно восстанавливает HP
	// (outOfCombatRegenFrac от MaxHP/с). Любой урон обнуляет таймер.
	noDamageT float64
}

type WorldZoneMsg struct {
	Index int     `json:"index"`
	Biome int     `json:"biome"`
	Name  string  `json:"name"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	W     float64 `json:"w"`
	H     float64 `json:"h"`
	Color uint32  `json:"color"`

	CornerRs [4]float64 `json:"cornerRs"` // радиус скругления каждого угла: [в-л, в-п, н-л, н-п]. 0 = прямые.

	SafeH  float64 `json:"safeH"`  // высота Safe Zone (низ зоны)
	WallTh float64 `json:"wallTh"` // толщина разделительной стены
	WallGap float64 `json:"wallGap"` // ширина прохода посередине стены

	Walls []WorldWallMsg `json:"walls"` // конфигурируемые стены зоны (локальные координаты)
}

// WorldWallMsg — стена-препятствие в локальных координатах зоны (0..W, 0..H).
type WorldWallMsg struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

type WorldPortalMsg struct {
	ID   int     `json:"id"`
	From int     `json:"from"`
	To   int     `json:"to"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	W    float64 `json:"w"`
	H    float64 `json:"h"`
}

type WorldMsg struct {
	Type    string           `json:"type"`
	Zones   []WorldZoneMsg   `json:"zones"`
	Portals []WorldPortalMsg `json:"portals"`

	// Волновой режим: клиент рисует круглую арену вместо зон/порталов.
	Wave      bool    `json:"wave,omitempty"`
	ArenaX    float64 `json:"arenaX,omitempty"`
	ArenaY    float64 `json:"arenaY,omitempty"`
	ArenaR    float64 `json:"arenaR,omitempty"`
	ArenaZone int     `json:"arenaZone,omitempty"` // зона-арена (тема фона/биом)
}

type TransferMsg struct {
	Type string `json:"type"`
	From int    `json:"from"`
	To   int    `json:"to"`
}

// CompassMsg — направление на ближайшего Fractal/Chaos-моба (петаль compass).
// A — угол (радианы, мир) от игрока к цели; D — дистанция.
type CompassMsg struct {
	Type string  `json:"type"`
	On   bool    `json:"on"`
	A    float64 `json:"a,omitempty"`
	D    float64 `json:"d,omitempty"`
}

type StateMsg struct {
	Type        string             `json:"type"`
	Players     []PlayerState      `json:"players"`
	Mobs        []MobState         `json:"mobs"`
	Petals      []PetalState       `json:"petals"`
	Ground      []GroundPetalState `json:"ground"`
	Projectiles []ProjectileState  `json:"projectiles"`
	Chains      []ChainState       `json:"chains"`

	// Wave bar (волновой режим): фаза и заполнение полос + ростер живых мобов.
	WavePhase  string            `json:"wavePhase,omitempty"` // spawn | hold
	WaveF1     float64           `json:"waveF1,omitempty"`    // слой 1: прогресс спавна 0..1
	WaveF2     float64           `json:"waveF2,omitempty"`    // слой 2: hold-таймер 0..1
	WaveNum    int               `json:"waveNum,omitempty"`
	WaveRoster []WaveRosterEntry `json:"waveRoster,omitempty"`
}

// ===== SecretProjectWave: типы волнового режима =====

// WaveMsg — события волнового режима: start/clear/pending/gameover.
type WaveMsg struct {
	Type    string  `json:"type"`
	Num     int     `json:"num"`
	Special string  `json:"special,omitempty"`
	State   string  `json:"state"`
	Break   float64 `json:"break,omitempty"`
}

// WaveRosterEntry — живые мобы волны по типу+редкости (для чипов под баром).
type WaveRosterEntry struct {
	Type   string `json:"type"`
	Name   string `json:"name"`
	Color  uint32 `json:"color"`
	Rarity int    `json:"rarity"`
	Count  int    `json:"count"`
}
