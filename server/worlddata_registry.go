package main

// worlddata_registry.go — подключение пакета internal/worlddata (порт из
// родительского SecretProject, адаптация 2.0). Типы и таблицы остаются под
// прежними идентификаторами package main, чтобы не менять места вызова.

import "secretproject/server/internal/worlddata"

// Типы-алиасы: определение живёт в worlddata, рантайм Wave работает со старыми именами.
type (
	MobType     = worlddata.MobType
	PetalKind   = worlddata.PetalKind
	PetalDef    = worlddata.PetalDef
	RareDropDef = worlddata.RareDropDef
)

// Таблицы мобов/петалей/дропов — единый каталог worlddata (47 петалей, ~48
// мобов, 16-строчная таблица редкостей дропа). Запись мобов — только через
// worlddata.RegisterMob.
var (
	mobTypes       = worlddata.MobTypes
	petalKinds     = worlddata.PetalKinds
	petalKindsPool = worlddata.PetalKindsPool
	mobDropKind    = worlddata.MobDropKind
	mobRareDrop    = worlddata.MobRareDrop
	mobRarityDrop  = worlddata.MobRarityDrop
)
