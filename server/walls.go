package main

import (
	"encoding/json"
	"os"
)

// walls.go — конфигурируемые стены-препятствия в отдельных зонах.
// Раскладка хранится в walls.json (локальные координаты ЗОНЫ: x от 0 до zoneW,
// y от 0 до zoneH, y растёт вниз). Стены блокируют и игроков, и мобов —
// тем самым они формируют "моб zone" внутри каждой зоны (лабиринты, коридоры,
// засады). Редактор: статичная страница editor.html + API /api/walls.

const wallLayoutFile = "walls.json"

// WallRect — прямоугольник стены в ЛОКАЛЬНЫХ координатах зоны.
type WallRect struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

// wallLayouts — раскладка стен по зонам (индекс зоны -> список стен).
var wallLayouts [][]WallRect

// wallRectsCache — стены зон в ГЛОБАЛЬНЫХ координатах (для коллизий).
// Кэшируются, чтобы не аллоцировать слайс на каждый вызов zoneWallRects()
// в тике (движение каждого моба/миньона). Инвалидируется в setWallLayout.
var wallRectsCache [][]Rect

// initWalls загружает раскладку из файла; если файла нет — сохраняет пустую
// (моб-зоны пока без отдельных стен, работает только разделительная стена
// между Safe Zone и Mob Zone). Раскладку можно менять через редактор.
func initWalls() {
	wallLayouts = make([][]WallRect, len(zones))
	wallRectsCache = make([][]Rect, len(zones))
	if !loadWalls() {
		saveWalls()
	}
	for i := 0; i < len(zones); i++ {
		wallRectsCache[i] = nil
	}
}

// applyDefaultWalls задавал стартовые раскладки стен; сейчас моб-зоны намеренно
// без отдельных стен (только разделительная между Safe и Mob Zone), поэтому
// функция пустая — стены можно добавлять через редактор /api/walls.
func applyDefaultWalls() {}

// normalizeWallRect приводит стену к валидному виду: в границах зоны,
// положительные размеры, минимальный размер.
func normalizeWallRect(r WallRect, z Zone) WallRect {
	if r.X < 0 {
		r.X = 0
	}
	if r.Y < 0 {
		r.Y = 0
	}
	if r.W < 40 {
		r.W = 40
	}
	if r.H < 40 {
		r.H = 40
	}
	if r.X+r.W > z.W {
		r.W = z.W - r.X
	}
	if r.Y+r.H > z.H {
		r.H = z.H - r.Y
	}
	return r
}

// setWallLayout заменяет раскладку стен зоны (с нормализацией).
func setWallLayout(zone int, rects []WallRect) {
	if zone < 0 || zone >= len(zones) {
		return
	}
	z := zones[zone]
	out := make([]WallRect, 0, len(rects))
	for _, r := range rects {
		r = normalizeWallRect(r, z)
		if r.W >= 40 && r.H >= 40 {
			out = append(out, r)
		}
	}
	wallLayouts[zone] = out
	// Стены изменились — сбрасываем кэш глобальных прямоугольников.
	if zone >= 0 && zone < len(wallRectsCache) {
		wallRectsCache[zone] = nil
	}
}

// zoneWallRects возвращает стены зоны в ГЛОБАЛЬНЫХ координатах (для коллизий).
// Результат кэшируется (см. wallRectsCache) и не должен модифицироваться.
func zoneWallRects(zone int) []Rect {
	if zone < 0 || zone >= len(zones) {
		return nil
	}
	if wallRectsCache[zone] == nil {
		z := zones[zone]
		out := make([]Rect, 0, len(wallLayouts[zone]))
		for _, w := range wallLayouts[zone] {
			out = append(out, Rect{X: z.X + w.X, Y: z.Y + w.Y, W: w.W, H: w.H})
		}
		wallRectsCache[zone] = out
	}
	return wallRectsCache[zone]
}

// portalPocketRect — «карман» безопасности вокруг ПОРТАЛА ВНИЗ в зоне второго
// уровня (этим порталом игрок возвращается на первый уровень; в L2 нет Safe
// Zone, и мобы иначе садятся прямо на портал). Мобы не могут войти в карман
// телом (кроме погони — см. updateMob), но продолжают агриться и бить с края.
// Карман тянется от нижнего края зоны вверх на portalPocketH, ширина 2×halfW
// вокруг центра портала.
const (
	portalPocketHalfW = 300.0
	portalPocketH     = 380.0
)

func portalPocketRect(zone int) (Rect, bool) {
	if zone < secondTierOffset || zone >= len(zones) {
		return Rect{}, false // только L2
	}
	z := zones[zone]
	for i := range portals {
		p := &portals[i]
		// Вертикальный портал ВНИЗ (To = zone-8) у нижнего края зоны.
		if p.From == zone && p.To == zone-secondTierOffset {
			return Rect{X: p.X - portalPocketHalfW, Y: z.Y + z.H - portalPocketH, W: portalPocketHalfW * 2, H: portalPocketH}, true
		}
	}
	return Rect{}, false
}

// loadWalls читает walls.json. Возвращает true, если файл прочитан.
func loadWalls() bool {
	data, err := os.ReadFile(wallLayoutFile)
	if err != nil {
		return false
	}
	var doc struct {
		Zones [][]WallRect `json:"zones"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return false
	}
	if len(doc.Zones) != len(zones) {
		return false
	}
	for i := range doc.Zones {
		setWallLayout(i, doc.Zones[i])
	}
	return true
}

// saveWalls записывает текущую раскладку в walls.json.
func saveWalls() error {
	doc := struct {
		Zones [][]WallRect `json:"zones"`
	}{Zones: wallLayouts}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(wallLayoutFile, data, 0o644)
}
