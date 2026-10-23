// Command genrarities генерирует client/js/rarities.js из пакета server/rarity.
// Запуск: go generate ./... (из каталога server/) либо вручную go run ./cmd/genrarities.
package main

import (
	"log"
	"os"
	"path/filepath"

	"secretproject/server/rarity"
)

func main() {
	out := filepath.Join("..", "client", "js", "rarities.js")
	if err := os.WriteFile(out, []byte(rarity.RenderJS()), 0o644); err != nil {
		log.Fatalf("genrarities: %v", err)
	}
	log.Printf("genrarities: wrote %s", filepath.ToSlash(out))
}
