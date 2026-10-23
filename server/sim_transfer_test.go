package main

import (
	"fmt"
	"testing"
)

func findPortal(from, to int) *Portal {
	for i := range portals {
		if portals[i].From == from && portals[i].To == to {
			return &portals[i]
		}
	}
	return nil
}

func TestSimTransferFlow(t *testing.T) {
	initWorld()
	dt := 1.0 / 20.0
	for i := 0; i < secondTierOffset; i++ {
		up := findPortal(i, i+secondTierOffset)
		down := findPortal(i+secondTierOffset, i)
		if up == nil || down == nil {
			t.Fatalf("zone %d missing pair", i)
		}

		// Голая комната БЕЗ gameLoop/autosave: тест сам драйвит updatePlayer.
		// NewRoom(20) здесь давал гонку (фоновый тик мутирует те же карты
		// параллельно с прямыми записями теста) и утекал горутины в другие тесты.
		r := newTestRoom()
		p := &Player{State: PlayerState{ID: "p", X: up.X, Y: up.Y + 50, Size: playerSize, Zone: i}, zone: i}
		r.players["p"] = p
		p.portalCD = 0
		for n := 0; n < 400; n++ {
			r.updatePlayer(p, dt)
			if p.zone != i {
				break
			}
		}
		fmt.Printf("zone %2d UP   -> pos=(%.0f,%.0f) zone=%d size=%.0f\n", i, p.State.X, p.State.Y, p.zone, p.State.Size)
		if p.zone != i+secondTierOffset {
			t.Fatalf("zone %d up-transfer failed: zone=%d", i, p.zone)
		}

		p.State.X = down.X
		p.State.Y = down.Y - 50
		p.zone = i + secondTierOffset
		p.State.Zone = i + secondTierOffset
		p.transferTo = -1
		p.transferT = 0
		p.portalCD = 0
		for n := 0; n < 400; n++ {
			r.updatePlayer(p, dt)
			if p.zone != i+secondTierOffset {
				break
			}
		}
		fmt.Printf("zone %2d DOWN -> pos=(%.0f,%.0f) zone=%d size=%.0f\n", i, p.State.X, p.State.Y, p.zone, p.State.Size)
		if p.zone != i {
			t.Fatalf("zone %d down-transfer failed: zone=%d", i, p.zone)
		}
	}
}

// Регрессия на портал-пинг-понг: после трансфера игрок обязан появляться ЗА
// пределами прямоугольника обратного портала, чтобы после окончания
// portalCooldown его не «засасывало» обратно.
func TestPortalLandingOutsideReturnRect(t *testing.T) {
	initWorld()
	dt := 1.0 / 20.0
	for i := 0; i < secondTierOffset; i++ {
		up := findPortal(i, i+secondTierOffset)
		down := findPortal(i+secondTierOffset, i)
		if up == nil || down == nil {
			t.Fatalf("zone %d missing pair", i)
		}

		// DOWN-трансфер (L2 -> L1): приземление в зоне i рядом с её up-порталом.
		p := &Player{State: PlayerState{ID: "p", X: up.X, Y: up.Y + 50, Size: playerSize, Zone: i}, zone: i}
		p.portalCD = 0
		r := newTestRoom()
		r.players["p"] = p
		r.placeAtPortal(p, i, i+secondTierOffset)
		if p.zone != i {
			t.Fatalf("zone %d down landing zone=%d, want %d", i, p.zone, i)
		}
		if z := portalAt(p.State.X, p.State.Y, p.State.Size); z != nil {
			t.Fatalf("zone %d down landing (%.0f,%.0f) inside portal %d->%d", i, p.State.X, p.State.Y, z.From, z.To)
		}
		p.portalCD = 0
		for n := 0; n < 300; n++ { // 15s: portalCD успел истечь
			r.updatePlayer(p, dt)
			if p.zone != i {
				t.Fatalf("zone %d down: re-triggered into zone %d after cooldown", i, p.zone)
			}
		}

		// UP-трансфер (L1 -> L2): приземление в зоне i+8 рядом с её down-порталом.
		q := &Player{State: PlayerState{ID: "q", X: down.X, Y: down.Y - 50, Size: playerSize, Zone: i + secondTierOffset}, zone: i + secondTierOffset}
		q.portalCD = 0
		r2 := newTestRoom()
		r2.players["q"] = q
		r2.placeAtPortal(q, i+secondTierOffset, i)
		if q.zone != i+secondTierOffset {
			t.Fatalf("zone %d up landing zone=%d", i, q.zone)
		}
		if z := portalAt(q.State.X, q.State.Y, q.State.Size); z != nil {
			t.Fatalf("zone %d up landing (%.0f,%.0f) inside portal %d->%d", i, q.State.X, q.State.Y, z.From, z.To)
		}
		q.portalCD = 0
		for n := 0; n < 300; n++ {
			r2.updatePlayer(q, dt)
			if q.zone != i+secondTierOffset {
				t.Fatalf("zone %d up: re-triggered into zone %d after cooldown", i, q.zone)
			}
		}
	}
}