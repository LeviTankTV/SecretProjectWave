// AUTO-GENERATED from server/rarity (go generate). DO NOT EDIT.
// Единый реестр редкостей клиента: имена, цвета, сила петаль, шансы крафта.

const rarityNames = ['Point', 'Line', 'Triangle', 'Square', 'Pentagon', 'Hexagon', 'Fractal', 'Chaos', 'Dot', 'Vector', 'Prism', 'Frame', 'Lattice', 'Spiral', 'Mirror', 'Origin'];
const rarityColors = ['#ecf0f1', '#2ecc71', '#f1c40f', '#e74c3c', '#3498db', '#1abc9c', '#9b59b6', '#ff2d95', '#bcc8cb', '#23a55a', '#d0a308', '#bd3d2e', '#2c7cb8', '#169a81', '#7d4794', '#cd2378'];
const RAR_MULT = [1, 1.5, 2.25, 3.375, 6, 12, 27, 81, 1.2247448713915892, 1.8371173070873836, 2.7556759606310757, 4.5, 8.485281374238571, 18, 46.76537180435969, 121.5];
const RARITY_DANGER = [1, 1.5, 2.25, 3.375, 5.0625, 7.59375, 11.390625, 17.0859375, 1.2247448713915892, 1.8371173070873836, 2.7556759606310757, 4.133513940946612, 6.200270911419919, 9.300406367049879, 13.950609550574818, 25.62890625];
const CRAFT_CHANCE_PCT = [20, 12.5, 7.5, 4, 2, 1, 1, 4, 20, 12.5, 7.5, 4, 2, 1, 1];

// Имя редкости по индексу; '?' при выходе за диапазон.
//
// Локализация: английское имя — только фолбэк. Живая подпись берётся из
// словаря клиента (ключ rarity.<индекс>, client/js/i18n.js), поэтому виды
// редкости в хотбаре/крафте/галерее русские в русском UI. Тестовые данные
// (const rarityNames выше) остаются основой сравнения — их не трогаем.
function rarityName(r) {
	const en = rarityNames[r];
	if (en == null) return '?';
	return (typeof window !== 'undefined' && typeof window.tr === 'function')
		? window.tr('rarity.' + r, en) : en;
}
