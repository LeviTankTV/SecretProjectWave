package main

import "math"

// absorb.go — Абсорб: уничтожение петалей инвентаря за опыт (XP).
// Полная противоположность крафта: петали (несколько видов/редкостей сразу)
// списываются навсегда, а игрок получает за них XP. Чем реже и сильнее петаль,
// тем больше опыта.

type absorbItem struct {
	Kind   string `json:"kind"`
	Rarity int    `json:"rarity"`
	Count  int    `json:"count"`
}

// Абсорб-цена петаля растёт в ГЕОМЕТРИЧЕСКОЙ ПРОГРЕССИИ по редкости:
// каждый шаг редкости даёт ровно absorbXPRatio × больше опыта
// (по умолчанию редкость +1 = вдвое больше XP). Cостав — «сила» вида
// (петаль без учёта редкости: BaseDMG*2 + BaseHP), от неё берём долю
// absorbXPCoeff и умножаем на 2^редкость.
const (
	absorbXPRatio = 2.0 // геометрический множитель за каждый шаг редкости
	absorbXPCoeff = 0.6 // доля «силы» за петаль базовой редкости (0)
	// absorbMaxKills — потолок цены ОДНОЙ петали в «убийствах моба той же
	// редкости» (порт родителя, XP_DESIGN §2.2): без него одна высокотировая
	// петаль стоила до 20 убийств — «абсорбить» было выгоднее, чем играть.
	absorbMaxKills = 10
)

// isSoulboundPetal — предмет-подарок, не являющийся ресурсом (порт родителя):
// ловушка trap и жетоны Ying/Yang не абсорбятся и не выбрасываются.
func isSoulboundPetal(kind string) bool {
	pk := petalKinds[kind]
	return pk.Trap || pk.Token
}

// petalStackXP — «сырая» цена петаля без потолка и штрафа (порт родителя):
// НОМИНАЛ вида из каталога, а не статы предмета — потенциал (Pot) множит
// HP/DMG, и без этого крафт -> абсорб давал бы больше XP за тот же вид
// (эксплойт §17.6). Номинал не зависит и от крафтового пола урона.
func petalStackXP(d PetalDef) float64 {
	rar := d.Rarity
	if rar < 0 {
		rar = 0
	}
	baseDMG, baseHP := d.DMG, d.MaxHP
	if kn, ok := petalKinds[d.Kind]; ok {
		baseDMG, baseHP = kn.BaseDMG, kn.BaseHP
	}
	base := (baseDMG*2 + baseHP) * absorbXPCoeff
	if base <= 0 {
		return 0
	}
	return base * math.Pow(absorbXPRatio, float64(rar))
}

// petalXPCapped — цена до штрафа тиров: «сырая», обрезанная потолком
// absorbMaxKills убийств своей редкости.
func petalXPCapped(d PetalDef) float64 {
	raw := petalStackXP(d)
	if raw <= 0 {
		return 0
	}
	rar := d.Rarity
	if rar < 0 {
		rar = 0
	}
	if capXP := float64(absorbMaxKills) * killXPRounded(rar); raw > capXP {
		raw = capXP
	}
	return raw
}

// petalXPFor — XP за уничтожение ОДНОГО петаля игроком уровня level: цена до
// штрафа × tierGap (тот же анти-фарм, что у убийства мобов). Минимум 1.
func petalXPFor(d PetalDef, level int) float64 {
	capped := petalXPCapped(d)
	if capped <= 0 {
		return 0
	}
	rar := d.Rarity
	if rar < 0 {
		rar = 0
	}
	return math.Max(1, math.Round(capped*tierGap(level, rar)))
}

// absorb уничтожает указанные петали из инвентаря и возвращает суммарный XP
// (считает только реально списанные, по доступному количеству в стаках).
func (sp *statelessProfile) absorb(items []absorbItem) float64 {
	var total float64
	for _, it := range items {
		if isSoulboundPetal(it.Kind) {
			continue // капкан/жетоны абсорбировать нельзя — молча пропускаем
		}
		i := sp.findStack(it.Kind, it.Rarity)
		if i < 0 || sp.inv[i].Count <= 0 || it.Count <= 0 {
			continue
		}
		n := it.Count
		if n > sp.inv[i].Count {
			n = sp.inv[i].Count
		}
		// Захватываем статы ДО списания: takeN может удалить стак из среза.
		def := sp.inv[i].PetalDef
		sp.takeN(it.Kind, it.Rarity, n)
		total += petalXPFor(def, sp.level) * float64(n)
	}
	return total
}
