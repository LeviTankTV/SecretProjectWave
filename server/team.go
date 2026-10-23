package main

// team.go — система команд/враждебности.
//
// Команда игрока (Player.Team):
//   0 — обычный (PvE): дружелюбен всем игрокам, враждебен мобам.
//   1 — враждебный (PvP-рого): враждебен ВСЕМ (игрокам и их петалям, мобам).
//   2 — команда мобов (только админ): враждебен игрокам, но НЕ взаимодействует
//       с мобами (не бьёт их, не получает урона, мобы не агрятся).
//
// Единая точка проверки враждебности — эти три функции.

// teamAreEnemies — враждебны ли две команды.
//   - разные команды -> враждебны;
//   - одна команда: враждебны только если обе == 1 (оба «рого»);
//   - команды 0 и 2 дружат внутри своей команды.
func teamAreEnemies(a, b int) bool {
	if a == b {
		return a == 1
	}
	return true
}

// playersAreEnemies — два игрока враждебны друг другу.
func playersAreEnemies(a, b *Player) bool {
	return teamAreEnemies(a.Team, b.Team)
}

// playerHostileToMob — игрок враждебен мобам (может бить их и получать урон).
// Команда 2 на стороне мобов: не враждебна.
func playerHostileToMob(team int) bool {
	return team != 2
}

// mobAggrosPlayer — моб агрится на игрока. Команда 2 невидима для мобов.
func mobAggrosPlayer(team int) bool {
	return team != 2
}
