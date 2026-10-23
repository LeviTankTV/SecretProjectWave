package main

import (
	crand "crypto/rand"
	"encoding/hex"
	"math"
	"math/rand"

	"secretproject/server/internal/worlddata"
)

// pot.go — «Потенциал» (см. Pot.md §17) у скрафченного петаля: два целых числа.
//
//	Pot — множитель HP и DMG, 100..cap. 100 = база (нет бонуса).
//	Eff — множитель пассивок/механик вида, 100..cap. 100 = база.
//
// Оба ролла делаются ТОЛЬКО в крафте и только на сервере: клиент эти числа не
// присылает, он их отображает. Ниже 100 не опускается никогда — крафт и так
// наказывает провалом (потеря 1..4 петалей, шансы до 1%).
const (
	potMaxPct     = 200  // абсолютный потолок обоих чисел
	potCapSafety  = 0.95 // 95% от отношения Power[r+1]/Power[r]
	potCapZones   = 7    // петели живут только на зонной лестнице Point..Chaos (0..7)
	maxCraftFrom  = 6    // Frac tal→Chaos ещё крафтится; из Chaos крафта нет
	potBandCommon = 0.60 // веса полос: обычный / хороший / отличный / годролл
	potBandGood   = 0.90
	potBandGreat  = 0.99
)

// ---- Грейды (Pot.md §17.1 v3, 20.10.2026) ----
//
// Раньше Pot был НЕПРЕРЫВНЫМ роллом 100..175 с четырьмя полосами, а верхние 10%
// ролла объявлялись «уникальными»: отдельный UID, отдельный стак, отдельный
// слой протокола. Итог — двадцать стаков одного вида, +33% рядом с «UNIQUE» и
// половина кода ради одного числа.
//
// Теперь грейд ДИСКРЕТНЫЙ: пять значений НАД базой, одно на предмет, и оно же
// умножает ВСЁ — и HP/DMG (potMult), и эффект пассивки (effMult). Второго числа
// (Eff) больше нет: Eff всегда равен Pot.
//
// Лестница (20.10.2026): UNIQUE → SUPERB → PRISTINE → MYTHIC → PARAGON.
// MYTHIC и PARAGON намеренно НЕ равны по шансу: 0.4% против 0.1%, иначе
// последняя ступень перестаёт быть событием.
var (
	// gradePct — множитель грейда в процентах (100 = база).
	gradePct = []int{100, 110, 125, 150, 175, 200}
	// gradeWeight — шансы выпадения в процентах: 72 / 19 / 7 / 1.5 / 0.4 / 0.1.
	gradeWeight = []float64{72, 19, 7, 1.5, 0.4, 0.1}
	// gradeName — подпись на карточке. База без подписи: «обычный» — не грейд.
	gradeName = []string{"", "UNIQUE", "SUPERB", "PRISTINE", "MYTHIC", "PARAGON"}
)

// potRand — источник случайности ролла. Переменная, а не прямой вызов
// rand.Float64: тесты подменяют её фиксированным значением и проверяют
// границы, cap и «pot=100 эквивалентен прежним статам» без флаки.
var potRand = rand.Float64

// gradeMaxFor — максимальный индекс грейда для полученной редкости.
//
// Ограничение по тиру — чтобы младший PARAGON не догнал следующую редкость:
// Point–Triangle (0..2) → UNIQUE, Square (3) → SUPERB, Pentagon/Hexagon (4..5)
// → PRISTINE, Fractal (6) → MYTHIC, Chaos (7) → PARAGON.
func gradeMaxFor(rarity int) int {
	switch {
	case rarity <= 2:
		return 1
	case rarity == 3:
		return 2
	case rarity <= 5:
		return 3
	case rarity == 6:
		return 4
	default:
		return len(gradePct) - 1
	}
}

// gradeOf — индекс грейда по проценту Pot. Негрейдовое (легаси) значение
// скругляется к ближайшему грейду — так старые предметы становятся обычными
// стаками без вайпа.
func gradeOf(pot int) int {
	best, bestD := 0, -1
	for i, p := range gradePct {
		d := pot - p
		if d < 0 {
			d = -d
		}
		if bestD < 0 || d < bestD {
			best, bestD = i, d
		}
	}
	return best
}

// snapPot — Pot приводится к дискретному грейду и к потолку ТИРА. Легаси-предметы
// с непрерывным роллом (100..175) при загрузке получают ближайший грейд — это и
// есть миграция на новую систему (UID при этом снимается: см. normalizePetalDef).
func snapPot(pot, rarity int) int {
	g := gradeOf(normPot(pot))
	if max := gradeMaxFor(rarity); g > max {
		g = max
	}
	return gradePct[g]
}

// gradeLabel — подпись грейда ("" для базы).
func gradeLabel(pot int) string {
	g := gradeOf(normPot(pot))
	if g <= 0 || g >= len(gradeName) {
		return ""
	}
	return gradeName[g]
}

// normPot/normEff — легаси-предметы (поле отсутствовало, прочиталось как 0)
// читаются как база 100. Ниже 100 не бывает: потенциал только бонус.
func normPot(p int) int {
	if p < 100 {
		return 100
	}
	return p
}

func normEff(e int) int {
	if e < 100 {
		return 100
	}
	return e
}

func potMult(p int) float64 { return float64(normPot(p)) / 100 }
func effMult(e int) float64 { return float64(normEff(e)) / 100 }

// potCap — максимальный Pot/Eff для ПОЛУЧЕННОЙ редкости r.
//
// Считается из живой кривой petalPower (её правит админка: petal.power), а не
// из таблицы. 95% оставляют зазор: максимум предыдущего тира не дотягивает до
// базы следующего. Для Chaos (и вообще верха лестницы) — 175: после Chaos идёт
// Dot, но это ДРУГАЯ лестница (истинные редкости — мобовские), её Power меньше
// Chaos в 66 раз, и формула «следующий/текущий» дала бы потолок 100%, то есть
// у самого дорогого тира потенциала не было бы вообще.
func potCap(r int) int {
	if r < 0 {
		r = 0
	}
	if r >= potCapZones || r >= len(petalPower)-1 {
		return potMaxPct
	}
	cur, next := petalPower[r], petalPower[r+1]
	if cur <= 0 || next <= cur { // немонотонная кривая — не отнимаем бонус
		return potMaxPct
	}
	// ВАЖНО: 100× перед отношением — potCapSafety это доля (0.95), а не
	// проценты; без множителя пол получается 1 и кап зажимается в 100.
	capPct := min(int(math.Floor(100*potCapSafety*next/cur)), potMaxPct)
	if capPct < 100 {
		capPct = 100
	}
	return capPct
}

// potThreshold — порог уникальности «верхние 10% роллов внутри тира».
//
// Значение в точке x=0.25 (граница «отличной» полосы) — но СТРОГО выше неё:
// ролл округляется до целого процента, и на младших тирах один процент
// «стоит» почти двух десятых x, из-за чего включение самой границы давало бы
// 12% уникальных вместо 10. Отсюда +1: 142→115, 168→123, 175→126.
func potThreshold(capPct int) int {
	if capPct <= 100 {
		return 100
	}
	return int(math.Floor(100+float64(capPct-100)/3)) + 1
}

// rollPctFor — один бросок в [100, capPct] по полосам 60/30/9/1%.
//
// [1.00..1.10) 60% · [1.10..1.25) 30% · [1.25..1.50) 9% · [1.50..1.75) 1%.
// Абстрактное пространство x сжимается в [1, cap/100] МАСШТАБИРОВАНИЕМ, а не
// clamp: при clamp все высокие роллы собрались бы в ровно cap и читались бы
// как баг. Средний результат на cap 175 ≈ 112.25%, на cap 142 ≈ 106.9% —
// тир по-прежнему доминирует над качеством.
func rollPctFor(rarity int) int {
	// Бросок по весам, ограниченный потолком ТИРА: недоступные грейды не могут
	// выпасть, а их вес пересчитывается на разрешённые (иначе младшие тиры молча
	// теряли бы часть распределения, и UNIQUE на Point был бы втрое реже).
	max := gradeMaxFor(rarity)
	total := 0.0
	for i := 0; i <= max; i++ {
		total += gradeWeight[i]
	}
	roll := potRand() * total
	acc := 0.0
	for i := 0; i <= max; i++ {
		acc += gradeWeight[i]
		if roll < acc {
			return gradePct[i]
		}
	}
	return gradePct[0]
}

// rollPot — бросок грейда для полученной редкости r. Возвращает процент
// (100/110/125/150/175/200) — Pot остаётся числом для статов, но принимает
// шесть значений вместо непрерывного диапазона.
func rollPot(r int) int { return rollPctFor(r) }

// rollEff — Эффект больше не отдельный ролл: один грейд на предмет умножает и
// статы, и пассивку. Функция оставлена для совместимости вызовов.
func rollEff(r int) int { return rollPot(r) }

// hasPassiveMechanic — у вида есть механика, которая считается от редкости
// (реген, хил, барьер, заводка, аура, радиусы и т.п.), а не от d.DMG/d.MaxHP.
func hasPassiveMechanic(kn worlddata.PetalKind) bool {
	return kn.Overdrive || kn.Shield || kn.Eye || kn.View || kn.Compass ||
		kn.Saturn || kn.MaxHP || kn.Guard || kn.Swift || kn.Spin || kn.Twirl ||
		kn.Slow || kn.Rune || kn.Split || kn.Regen || kn.Tear
}

// potEligible — роллится ли Потенциал (HP/DMG) у вида.
//
// Правило: у вида есть реальный боевой стат и НЕТ чисто-пассивной механики.
// У видов с пассивкой Pot не роллится не потому, что он «не влияет на тело», а
// потому, что их сила описана отдельным числом Eff (см. §17.2): иначе один и
// тот же предмет получал бы два бонуса за одно и то же.
func potEligible(kind string) bool {
	kn, ok := worlddata.PetalKinds[kind]
	if !ok || kn.BaseDMG <= 0 || kn.BaseHP <= 0 {
		return false
	}
	if kn.Trap || kn.Token || kn.Passive || kn.MinionType != "" {
		return false
	}
	return !hasPassiveMechanic(kn)
}

// effEligible — роллится ли Эффект у вида: нужен хотя бы один пассивный флаг
// (или миньон-спавнер). У чистых дд-видов Eff = 100 — ложного блеска нет.
func effEligible(kind string) bool {
	kn, ok := worlddata.PetalKinds[kind]
	if !ok || kn.Trap || kn.Token || kn.Passive {
		return false
	}
	return kn.MinionType != "" || hasPassiveMechanic(kn)
}

// newItemUID — стабильный ID уникального предмета (Pot.md §17.3).
//
// crypto/rand, а не math/rand: по UID адресуются операции над конкретным
// предметом (equip/drop/move), и угадываемый UID позволил бы трогать чужой
// предмет в будущей торговле. 8 байт = 16 hex-символов, коллизий на объёмах
// инвентаря нет.
func newItemUID() string {
	b := make([]byte, 8)
	if _, err := crand.Read(b); err != nil {
		// Практически недостижимо (crypto/rand падает только при отсутствии
		// энтропии); Unique всё равно должен быть уникальным в рамках процесса.
		for i := range b {
			b[i] = byte(rand.Intn(256))
		}
	}
	return "itm_" + hex.EncodeToString(b)
}

// isUniquePot — попадает ли ролл в уникальный срез тира (§17.3): порог
// считается от потолка РЕДКОСТИ (потолки разные: 142/168/175), а не
// фиксированным числом. Сама уникальность определяется только потенциалом.
func isUniquePot(pot, rarity int) bool {
	return normPot(pot) >= potThreshold(potCap(rarity))
}

// sameStackDef — лежат ли два предмета в один стак.
//
// Ключ стака — (вид, редкость, ГРЕЙД). Грейд теперь дискретный (см. gradePct),
// поэтому вариантов 3–5 вместо десятков непрерывных роллов. Eff в ключ не входит:
// он больше не роллится отдельно и всегда равен Pot.
// UID-ветка оставлена только для легаси-предметов, которые ещё не прошли
// normalizePetalDef (миграция снимает UID и скругляет Pot к грейду).
func sameStackDef(a, b PetalDef) bool {
	if a.UID != "" || b.UID != "" {
		return a.UID == b.UID
	}
	return a.Kind == b.Kind && a.Rarity == b.Rarity &&
		normPot(a.Pot) == normPot(b.Pot)
}
