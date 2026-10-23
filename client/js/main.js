// main.js - клиент игры.
// Рисует мир (зоны, порталы, игроков) через PixiJS, ведёт камеру за игроком,
// показывает миникарту (только текущую зону), надпись зоны и экран перехода.
// Вся игровая логика (движение, зоны, телепорты) живёт на сервере.

// ----- DOM элементы -----
const loginScreen = document.getElementById('loginScreen');
const registerScreen = document.getElementById('registerScreen');
const menu = document.getElementById('menu');
const loginNick = document.getElementById('loginNick');
const loginPass = document.getElementById('loginPass');
const loginBtn = document.getElementById('loginBtn');
const authError = document.getElementById('authError');
const regNick = document.getElementById('regNick');
const regPass = document.getElementById('regPass');
const regBtn = document.getElementById('regBtn');
const regError = document.getElementById('regError');
const playBtn = document.getElementById('playBtn');
const errorBox = document.getElementById('errorBox');
const zoneLabel = document.getElementById('zoneLabel');
const transferEl = document.getElementById('transfer');
const transferText = document.getElementById('transferText');
const minimapCanvas = document.getElementById('minimap');
const mctx = minimapCanvas.getContext('2d');
const hpBar = document.getElementById('hpBar');
const hpFill = document.getElementById('hpFill');
const shieldBar = document.getElementById('shieldBar');
const shieldFill = document.getElementById('shieldFill');
const hpText = document.getElementById('hpText');
const xpBar = document.getElementById('xpBar');
const xpFill = document.getElementById('xpFill');
const xpText = document.getElementById('xpText');
const deathEl = document.getElementById('death');
const loadingEl = document.getElementById('loading');
const loadingText = document.getElementById('loadingText');
const chatButton = document.getElementById('chatButton');
const chatWindow = document.getElementById('chatWindow');
const chatLog = document.getElementById('chatLog');
const chatInput = document.getElementById('chatInput');
let chatOpen = false;
let authToken = localStorage.getItem('authToken') || '';

// Цвет тега зоны в чате: у каждой зоны свой цвет (Point…Chaos, II-версии, TestRoom).
// Индекс совпадает с массивом зон, приходящим от сервера (msg.zones).
const ZONE_TAG_COLORS = [
    '#6fc3ff', '#2ecc71', '#f1c40f', '#e74c3c', '#3498db', '#1abc9c', '#9b59b6', '#ff2d95',
    '#8fd8ff', '#58d68d', '#f7dc6f', '#f1948a', '#5dade2', '#48c9b0', '#af7ac5', '#ff6bb0',
    '#aeb6bf',
];
// zoneTagColor возвращает цвет зоны по её имени (или null, если зона неизвестна).
function zoneTagColor(zoneName) {
    if (!zoneName || !zones || zones.length === 0) return null;
    const i = zones.findIndex(z => z && z.name === zoneName);
    if (i < 0 || i >= ZONE_TAG_COLORS.length) return null;
    return ZONE_TAG_COLORS[i];
}

// Анти-спам input при простое (АФК): чтобы не слать серверу ~60 пустых
// сообщений в секунду, запоминаем последнее отправленное состояние и шлём
// новое только при его изменении (или раз в 250мс).
let lastInput = null;
let lastInputT = 0;

// Игровое состояние (объявлено ЗДЕСЬ, чтобы избежать TDZ-ошибки:
// showScreen('menu') и обработчики логина обращаются к ним рано).
let myNickname = '';
let myLevel = 0;
let myXP = 0;
let myXPMax = 1;
let myInventory = []; // петали-предметы игрока (из state)
let myHotbars = [];   // две линейки хотбара (из state)
let myActiveBar = 0;  // активная линейка
// Маска двойного клика при «клин = надеть в первый свободный слот»: после
// первого клика петаль уже уехала в хотбар, повторный щелчок в том же окне
// игнорируем, чтобы не отправлять второй move одного и того же стака.
let inventoryEquipClickTs = 0;

// ----- Цвета в меню (для выбора круга) -----
const palette = [0xff5252, 0xff9800, 0xffeb3b, 0x2ecc71, 0x3498db, 0x9b59b6, 0xffffff];
// Выбранный цвет СОХРАНЯЕТСЯ всегда (localStorage) — переживает резет
// логин/сессию; на сервер уходит с 'hello' при входе в мир.
let chosenColor = palette[0];

// Персональный цвет для отдельного ника (mediumspringgreen).
const SPECIAL_COLORS = { rumeru: 0x00fa9a };

// Восстановление сохранённого цвета (localStorage) ДО построения палитры —
// переживает резет акаунта/БД: цвет игрока остаётся именно каким выбрали.
(function restoreChosenColor() {
    const saved = parseInt(localStorage.getItem('playerColor'), 10);
    if (!isNaN(saved)) {
        const ok = palette.includes(saved) || Object.values(SPECIAL_COLORS).includes(saved);
        if (ok) chosenColor = saved;
    }
})();
let specialColorAdded = false;
// saveChosenColor — выбор цвета всегда с сохранением в localStorage.
function saveChosenColor(c) {
    chosenColor = c;
    try { localStorage.setItem('playerColor', String(c)); } catch (e) {}
}
function ensureSpecialColor() {
    const c = SPECIAL_COLORS[myNickname];
    if (specialColorAdded || !c) return;
    specialColorAdded = true;
    const dot = document.createElement('div');
    dot.className = 'color-dot';
    dot.style.background = '#' + c.toString(16).padStart(6, '0');
    dot.addEventListener('click', () => {
        saveChosenColor(c);
        colorRow.querySelectorAll('.color-dot').forEach(d => d.classList.remove('selected'));
        dot.classList.add('selected');
        updatePlayerAvatar();
    });
    colorRow.appendChild(dot);
    // Если этот special-цвет был сохранён ранее — подсветим сразу.
    if (chosenColor === c) {
        dot.classList.add('selected');
        colorRow.querySelectorAll('.color-dot').forEach(d => {
            if (d !== dot) d.classList.remove('selected');
        });
    }
}

const colorRow = document.getElementById('colorRow');
palette.forEach((c, i) => {
    const dot = document.createElement('div');
    dot.className = 'color-dot' + (chosenColor === c ? ' selected' : '');
    dot.style.background = '#' + c.toString(16).padStart(6, '0');
    dot.addEventListener('click', () => {
        saveChosenColor(c);
        colorRow.querySelectorAll('.color-dot').forEach(d => d.classList.remove('selected'));
        dot.classList.add('selected');
        updatePlayerAvatar();
    });
    colorRow.appendChild(dot);
});

// Обновляет кружок-аватар игрока (сверху слева) под выбранный цвет игрока.
function updatePlayerAvatar() {
    const av = document.getElementById('playerAvatar');
    const dot = document.getElementById('playerAvatarDot');
    const hex = '#' + (chosenColor || palette[0]).toString(16).padStart(6, '0');
    if (av) {
        av.style.background = `radial-gradient(circle at 35% 30%, rgba(255,255,255,0.4), rgba(255,255,255,0) 60%), linear-gradient(135deg, ${hex}, ${hex})`;
        av.style.borderColor = hex;
        av.style.boxShadow = `0 0 12px ${hex}, inset 0 2px 6px rgba(255,255,255,0.3)`;
    }
    if (dot) {
        dot.style.background = `radial-gradient(circle at 35% 30%, #ffffff, ${hex})`;
        dot.style.boxShadow = `0 0 8px ${hex}`;
    }
    // Аватар в профиле главного меню: цвет меняется плавно (CSS-transition).
    const mAv = document.getElementById('menuAvatar');
    const mFill = document.getElementById('menuAvatarFill');
    if (mAv) {
        mAv.style.borderColor = hex;
        mAv.style.boxShadow = `0 0 14px ${hex}, inset 0 2px 6px rgba(255,255,255,0.3)`;
    }
    if (mFill) mFill.style.backgroundColor = hex;
}

// ----- Авторизация -----
// Если активен экран меню/входа — скрываем игровой мир (он за меню не виден),
// экономя GPU, но НЕ останавливаем тикер, чтобы не было «чёрного экрана».
function syncTicker() {
    const inGame = !menu.classList.contains('active') &&
                   !loginScreen.classList.contains('active') &&
                   !registerScreen.classList.contains('active') &&
                   deathEl && deathEl.style.display !== 'block';
    if (world) world.visible = inGame;
    // Скрываем игровой канвас целиком, когда не в игре (меню, регистрация, вход).
    const gameEl = document.getElementById('game');
    if (gameEl) gameEl.style.display = inGame ? 'block' : 'none';
    // Кнопка внутриигрового инвентаря и крафта: только в игре.
    const gib = document.getElementById('gameInvBtn');
    if (gib) gib.style.display = inGame ? 'flex' : 'none';
    const gcb = document.getElementById('gameCraftBtn');
    if (gcb) gcb.style.display = inGame ? 'flex' : 'none';
    // Игровой HUD (ник/HP, шкала опыта, хотбар) виден только в игре.
    const hudIds = ['playerHud', 'xpBar', 'gameHotbar', 'bossHud'];
    for (const id of hudIds) {
        const el = document.getElementById(id);
        if (el) el.style.display = inGame ? '' : 'none';
    }
    // Онлайн-счётчик — только в главном меню, в игре скрыт.
    const onlineBoxEl = document.getElementById('onlineBox');
    if (onlineBoxEl) onlineBoxEl.style.display = (inGame || !authToken) ? 'none' : 'block';
    if (!inGame) closeGameInv();
    if (app) {
        if (inGame) {
            if (app.ticker.stopped) app.ticker.start();
        }
        // Не останавливаем тикер в меню: просто не показываем мир (world.visible=false).
    }
}

function showScreen(name) {
    loginScreen.classList.remove('active');
    registerScreen.classList.remove('active');
    menu.classList.remove('active');
    if (name === 'login') loginScreen.classList.add('active');
    else if (name === 'register') registerScreen.classList.add('active');
    else if (name === 'menu') {
        menu.classList.add('active');
        updateMenuLevel();
        closeGameInv();
    }
    syncTicker();
}

// ----- Внутриигровой инвентарь -----
let gameInvOpen = false;
function toggleGameInv() {
    gameInvOpen = !gameInvOpen;
    const panel = document.getElementById('gameInvPanel');
    if (panel) panel.style.display = gameInvOpen ? 'flex' : 'none';
    if (gameInvOpen) updateGameInvPanel(true);
}
function closeGameInv() {
    gameInvOpen = false;
    const panel = document.getElementById('gameInvPanel');
    if (panel) panel.style.display = 'none';
}

// Сигнатура контента панели: инвентарь + активная линейка.
function gameInvSig() {
    let s = 'inv:' + myInventory.length + '|q:' + (window.__gameInvQ || '');
    for (const it of myInventory) s += ',' + (it.kind || '') + ':' + (it.rarity || 0) + ':' + (it.count || 1);
    return s;
}
let lastGameInvSig = '';

// Рисует панель: сетку инвентаря (с поиском по названию). Хотбар-полосу УБРАЛИ —
// игрок переставляет настоящий хотбар снизу-центр экрана (tooltip больше не
// перерывается ею).
function updateGameInvPanel(force) {
    if (!gameInvOpen) return;
    const sig = gameInvSig();
    if (!force && sig === lastGameInvSig) return;
    lastGameInvSig = sig;
    const body = document.getElementById('gameInvBody');
    if (!body) return;
    body.innerHTML = '';
    const q = (window.__gameInvQ || '').trim().toLowerCase();
    const items = (myInventory || []).filter((it) => {
        if (!q) return true;
        const nm = (it.name || kindName(it.kind) || '').toLowerCase();
        return nm.indexOf(q) !== -1;
    });
    if (!items.length) {
        const e = document.createElement('div');
        e.id = 'gameInvEmpty';
        e.textContent = q ? `No shapes match "${window.__gameInvQ}"` : 'No shapes in inventory';
        body.appendChild(e);
        return;
    }
    for (const it of items) {
        body.appendChild(buildInvStackSlot(it));
    }
}

// Слот-стак инвентаря (drag + клик надеть + ПКМ выброс) — как в меню.
function buildInvStackSlot(it) {
    const slot = document.createElement('div');
    slot.className = 'inv-slot';
    slot.draggable = true;
    const col = rarityColors[Math.min(it.rarity || 0, rarityColors.length - 1)];
    attachPetalTip(slot, it);
    const shape = document.createElement('div');
    shape.className = 'petal-shape';
    shape.innerHTML = petalSVG(it.kind, col);
    slot.appendChild(shape);
    const nm = document.createElement('div');
    nm.className = 'inv-name';
    nm.textContent = it.name || kindName(it.kind);
    slot.appendChild(nm);
    if ((it.count || 1) > 1) {
        const badge = document.createElement('div');
        badge.className = 'cat-count';
        badge.textContent = '×' + (it.count || 1);
        slot.appendChild(badge);
    }
    slot.addEventListener('contextmenu', (ev) => {
        ev.preventDefault();
        dropPetal({ src: 'inv', kind: it.kind || '', rarity: it.rarity || 0 });
    });
    slot.addEventListener('click', () => equipToFree(it.kind || '', it.rarity || 0));
    slot.addEventListener('dragstart', (ev) => {
        ev.dataTransfer.setData('text/plain', JSON.stringify({ src: 'inv', kind: it.kind || '', rarity: it.rarity || 0 }));
        ev.dataTransfer.effectAllowed = 'move';
        slot.classList.add('dragging');
    });
    slot.addEventListener('dragend', () => slot.classList.remove('dragging'));
    return slot;
}

// Слот хотбара (drop-цель + drag-источник + dblclick снять) — копия игрового.
function buildHotbarSlotLite(bar, i, it) {
    const slot = document.createElement('div');
    slot.className = it && it.id ? 'inv-slot hotbar-slot' : 'inv-slot hotbar-slot empty-slot';
    const col = (it && it.id) ? rarityColors[Math.min(it.rarity || 0, rarityColors.length - 1)] : null;
    if (it && it.id) attachPetalTip(slot, it);
    if (it && it.id) {
        const shape = document.createElement('div');
        shape.className = 'petal-shape';
        shape.innerHTML = petalSVG(it.kind, col);
        slot.appendChild(shape);
        const nm = document.createElement('div');
        nm.className = 'inv-name';
        nm.textContent = it.name || kindName(it.kind);
        slot.appendChild(nm);
    }
    slot.addEventListener('dragover', (ev) => { ev.preventDefault(); });
    slot.addEventListener('drop', (ev) => {
        ev.preventDefault();
        let data;
        try { data = JSON.parse(ev.dataTransfer.getData('text/plain')); } catch (e) { data = null; }
        if (!data) return;
        const from = (data.src === 'hot')
            ? { src: 'hot', bar: data.bar, slot: data.slot }
            : { src: 'inv', kind: data.kind, rarity: data.rarity };
        movePetal(from, bar, i);
        setTimeout(() => updateGameInvPanel(true), 60);
    });
    if (it && it.id) {
        slot.draggable = true;
        slot.addEventListener('dragstart', (ev) => {
            ev.dataTransfer.setData('text/plain', JSON.stringify({ src: 'hot', bar: bar, slot: i, kind: it.kind || '', rarity: it.rarity || 0 }));
            ev.dataTransfer.effectAllowed = 'move';
            slot.classList.add('dragging');
        });
        slot.addEventListener('dragend', () => slot.classList.remove('dragging'));
        slot.addEventListener('dblclick', (ev) => { ev.preventDefault(); unequipPetal(bar, i); setTimeout(() => updateGameInvPanel(true), 120); });
    }
    return slot;
}

// ----- Плавный переход на чёрный экран "Loading..." -----
function showLoading(text) {
    if (loadingText) loadingText.textContent = text || 'Loading...';
    if (loadingEl) loadingEl.classList.add('show');
}
function hideLoading() {
    if (loadingEl) loadingEl.classList.remove('show');
}

// Профиль в главном меню: ник в шапке и в строке "Logged in as".
function updateMenuProfile() {
    const nick = myNickname || '-';
    const n1 = document.getElementById('menuProfileNick');
    const n2 = document.getElementById('menuProfileNick2');
    if (n1 && n1.textContent !== nick) n1.textContent = nick;
    if (n2 && n2.textContent !== nick) n2.textContent = nick;
    updatePlayerAvatar();
}

function updateMenuLevel() {
    updateMenuProfile();
    const el = document.getElementById('menuLevel');
    if (el) el.textContent = `Lv. ${myLevel} — ${Math.floor(myXP)} / ${Math.floor(myXPMax)} XP`;
    const fill = document.getElementById('menuXpFill');
    if (fill) {
        const pct = Math.max(0, Math.min(100, (myXP / myXPMax) * 100));
        fill.style.width = pct + '%';
    }
    if (inventoryDirty()) {
        renderMenuInventory();
        lastInvSig = inventorySig();
    }
    if (hotbarDirty()) {
        renderMenuHotbar();
        renderGameHotbar();
        lastHotbarSig = hotbarSig();
    }
}

// Сигнатура инвентаря, чтобы не перестраивать DOM каждый тик.
let lastInvSig = '';
function inventorySig() {
    let s = '';
    for (const it of (myInventory || [])) s += ',' + (it.kind || '') + ':' + (it.rarity || 0) + 'x' + (it.count || 1);
    return s;
}
function inventoryDirty() { return inventorySig() !== lastInvSig; }

// Цвет редкости для картинки петаля — единый реестр js/rarities.js
// (автогенерация из server/rarity, 16 редкостей Point..Origin).

// рисует в меню петали из инвентаря (картинка зависит от редкости).
// Поддерживает перетаскивание (drag&drop) в хотбар и выброс за XP.
function renderMenuInventory() {
    const box = document.getElementById('menuInventory');
    if (!box) return;
    box.innerHTML = '';
    const q = (invSearchQuery || '').toLowerCase();
    let inv = (myInventory || []).filter((it) => {
        if (!q) return true;
        const lbl = (it.name || kindName(it.kind || '')) + ' ' + rarityName(it.rarity || 0) + ' ' + (it.kind || '');
        return lbl.toLowerCase().includes(q);
    });
    if (inv.length === 0) {
        const e = document.createElement('div');
        e.className = 'inv-empty';
        if (q) {
            const tx = document.createElement('div');
            tx.style.color = '#8a93a6';
            tx.style.fontSize = '12px';
            tx.textContent = 'No shapes match "' + invSearchQuery + '"';
            e.appendChild(tx);
        }
        box.appendChild(e);
        return;
    }
    const total = Math.max(6, inv.length);
    for (let i = 0; i < total; i++) {
        if (i < inv.length) {
            const it = inv[i];
            const slot = document.createElement('div');
            slot.className = 'inv-slot';
            slot.draggable = true;
            slot.dataset.src = 'inv';
            slot.dataset.kind = it.kind || '';
            slot.dataset.rarity = String(it.rarity || 0);
            const col = rarityColors[Math.min(it.rarity || 0, rarityColors.length - 1)];
            const cnt = it.count || 1;
            slot.title = '';
            attachPetalTip(slot, it);
            const shape = document.createElement('div');
            shape.className = 'petal-shape';
            shape.innerHTML = petalSVG(it.kind, col);
            slot.appendChild(shape);
            const nm = document.createElement('div');
            nm.className = 'inv-name';
            nm.textContent = it.name || kindName(it.kind);
            slot.appendChild(nm);
            if (cnt > 1) {
                const badge = document.createElement('div');
                badge.className = 'cat-count';
                badge.textContent = '×' + cnt;
                slot.appendChild(badge);
            }
            // Грейд (Pot): золотой чип в углу карточки — UNIQUE..PARAGON.
            const grade = it.pot > 100 ? gradeLabelOf(it.pot).split(' ×')[0] : '';
            if (grade) {
                const gb = document.createElement('div');
                gb.textContent = grade;
                gb.style.cssText = 'position:absolute;top:2px;left:2px;font-size:8px;font-weight:700;'
                    + 'color:#1a1208;background:linear-gradient(135deg,#ffd54f,#ffb300);'
                    + 'border-radius:4px;padding:1px 4px;letter-spacing:.4px;pointer-events:none;';
                slot.appendChild(gb);
            }
            // Выброс за XP (ПКМ)
            slot.addEventListener('contextmenu', (ev) => {
                ev.preventDefault();
                dropPetal({ src: 'inv', kind: it.kind || '', rarity: it.rarity || 0 });
            });
            // Клик — мгновенно надеть в первый свободный слот хотбара (удобно
            // быстро собирать линейки). Двойной щелчок не «унесёт» две петали:
            // второй клик в том же окне игнорируется.
            slot.addEventListener('click', (ev) => {
                ev.preventDefault();
                const now = Date.now();
                if (now - inventoryEquipClickTs < 300) return;
                inventoryEquipClickTs = now;
                equipToFree(it.kind || '', it.rarity || 0);
            });
            // Drag&drop
            slot.addEventListener('dragstart', (ev) => {
                ev.stopPropagation();
                ev.dataTransfer.setData('text/plain', JSON.stringify({ src: 'inv', kind: it.kind || '', rarity: it.rarity || 0 }));
                ev.dataTransfer.effectAllowed = 'move';
                slot.classList.add('dragging');
            });
            slot.addEventListener('dragend', () => slot.classList.remove('dragging'));
            box.appendChild(slot);
        } else {
            const e = document.createElement('div');
            e.className = 'inv-empty';
            box.appendChild(e);
        }
    }
}

// Имя вида по ключу.
function kindName(kind) {
    const names = {
        oval: 'Oval', thorns: 'Thorns', rock: 'Rock', ball: 'Ball',
        orb: 'Orb', pebble: 'Pebble', shard: 'Shard', cog: 'Cog',
        blade: 'Blade', viper: 'Viper', cube: 'Cube', bulwark: 'Bulwark',
        penta: 'Penta', shield: 'Shield', hex: 'Hex', hiveling: 'Hiveling',
        fractalite: 'Fractalite', spark: 'Spark', fragwall: 'Fragwall',
        void: 'Void', entropy: 'Entropy', singularity: 'Singularity',
        tear: 'Tear', line: 'Line', compass: 'Compass',
        saturn: 'Saturn', rune: 'Rune', eye: 'Eye', spike: 'Spike',
        rhombus: 'Rhombus', twirl: 'Twirl',
    };
    return names[kind] || (kind ? kind.charAt(0).toUpperCase() + kind.slice(1) : 'Oval');
}

// Рисует хотбар в меню (2 линейки, активная подсвечена). Слоты принимают
// перетаскивание из инвентаря и между линейками; двойной клик — снять;
// ПКМ — выбросить за XP.
// Рисует хотбар в меню: показываем ТОЛЬКО активную линейку (X переключает 1/2).
// Слоты принимают перетаскивание из инвентаря и между линейками; двойной клик —
// снять; ПКМ — выбросить за XP.
// Дешёвая сигнатура активной линейки, чтобы не перерисовывать DOM каждый тик.
let lastHotbarSig = '';
function hotbarSig() {
    const list = (myHotbars && myHotbars[myActiveBar]) ? myHotbars[myActiveBar] : [];
    let s = myActiveBar + '|' + list.length;
    for (const it of list) s += ',' + (it.id || '');
    return s;
}
function hotbarDirty() { return hotbarSig() !== lastHotbarSig; }

function renderMenuHotbar() {
    const wrap = document.getElementById('menuHotbarBar');
    const label = document.getElementById('menuHotbarLabel');
    if (wrap) {
        wrap.classList.add('active');
        wrap.innerHTML = '';
        const bar = myActiveBar;
        const list = (myHotbars && myHotbars[bar]) ? myHotbars[bar] : [];
        const cap = petalCapDisplay(bar);
        for (let i = 0; i < cap; i++) {
            const slot = document.createElement('div');
            slot.className = 'inv-slot hotbar-slot';
            slot.dataset.bar = String(bar);
            slot.dataset.slot = String(i);
            const it = (i < list.length) ? list[i] : null;
            if (!it || !it.id) slot.classList.add('empty-slot');
            // Drop-цель: перенос/мену в этот слот.
            slot.addEventListener('dragover', (ev) => { ev.preventDefault(); ev.dataTransfer.dropEffect = 'move'; });
            slot.addEventListener('drop', (ev) => {
                ev.preventDefault();
                ev.stopPropagation();
                let data;
                try { data = JSON.parse(ev.dataTransfer.getData('text/plain')); }
                catch (e) { data = null; }
                if (!data) return;
                if (data.src === 'hot') hotDropAccepted = true;
                // Принимаем из инвентаря или из другого хотбара — единый move.
                const from = (data.src === 'hot')
                    ? { src: 'hot', bar: data.bar, slot: data.slot }
                    : { src: 'inv', kind: data.kind, rarity: data.rarity };
                movePetal(from, bar, i);
            });
            // Сам слот можно тащить в другой слот/инвентарь.
            slot.setAttribute('draggable', String(!!(it && it.id)));
            slot.addEventListener('dragstart', (ev) => {
                if (!it) { ev.preventDefault(); return; }
                ev.stopPropagation();
                hotDropAccepted = false;
                ev.dataTransfer.setData('text/plain', JSON.stringify({ src: 'hot', bar: bar, slot: i, kind: it.kind || '', rarity: it.rarity || 0 }));
                ev.dataTransfer.effectAllowed = 'move';
                slot.classList.add('dragging');
            });
            slot.addEventListener('dragend', (ev) => {
                slot.classList.remove('dragging');
                // Бросили петаль куда-то, кроме хотбара/инвентаря → снять в запас.
                if (it && it.id && !hotDropAccepted) unequipPetal(bar, i);
            });

            if (it && it.id) {
                const col = rarityColors[Math.min(it.rarity || 0, rarityColors.length - 1)];
                attachPetalTip(slot, it);
                const shape = document.createElement('div');
                shape.className = 'petal-shape';
                shape.innerHTML = petalSVG(it.kind, col);
                slot.appendChild(shape);
                const nm = document.createElement('div');
                nm.className = 'inv-name';
                nm.textContent = it.name || kindName(it.kind);
                slot.appendChild(nm);
                slot.addEventListener('dblclick', (ev) => { ev.preventDefault(); unequipPetal(bar, i); });
                slot.addEventListener('contextmenu', (ev) => { ev.preventDefault(); dropPetal({ src: 'hot', bar: bar, slot: i }); });
                // Shift+наведение — снять петаль в инвентарь. Срабатывает один раз
                // за наведение (guard), сбрасывается при уходе мыши или отпускании Shift.
                slot.addEventListener('mouseenter', () => { shiftUnequipArmed = null; });
                slot.addEventListener('mouseleave', () => { shiftUnequipArmed = null; });
                slot.addEventListener('mousemove', (ev) => {
                    if (!ev.shiftKey) { shiftUnequipArmed = null; return; }
                    const key = bar + '_' + i;
                    if (shiftUnequipArmed !== key) {
                        shiftUnequipArmed = key;
                        unequipPetal(bar, i);
                    }
                });
            }
            wrap.appendChild(slot);
        }
    }
    if (label) label.textContent = `Hotbar ${myActiveBar + 1}`;
}

// Петали в хотбаре для показа: размер линейки = кап уровня (как на сервере).
// Важно: НЕ выводить больше слотов, чем позволяет сервер, иначе клик в лишний
// слот улетит в clamp и не отразится (например, «6-й слот не меняется»).
function petalCapDisplay(bar) {
    return petalCapForLevel(myLevel);
}

// Кап петалей по уровню — зеркало серверного petalCapForLevel.
function petalCapForLevel(level) {
    if (level >= 50) return 10;
    if (level >= 25) return 9;
    if (level >= 16) return 8;
    if (level >= 10) return 7;
    if (level >= 5) return 6;
    return 5;
}

// Надевает предмет из стака в первый свободный слот активного хотбара.
function equipToFree(kind, rarity) {
    const bar = myActiveBar;
    const list = (myHotbars && myHotbars[bar]) ? myHotbars[bar] : [];
    const cap = petalCapDisplay(bar);
    for (let i = 0; i < cap; i++) {
        const it = (i < list.length) ? list[i] : null;
        if (!it || !it.id) {
            movePetal({ src: 'inv', kind: kind, rarity: rarity }, bar, i);
            return;
        }
    }
}

// Рисует хотбар в игре (низ экрана) — активная линейка. Слоты можно таскать друг
// в друга (обмен местами) и сюда же можно кидать петали из инвентаря в меню.
let gameHotbarVeils = {};
function renderGameHotbar() {
    const wrap = document.getElementById('gameHotbarBar');
    const label = document.getElementById('gameHotbarLabel');
    if (!wrap) return;
    wrap.innerHTML = '';
    gameHotbarVeils = {};
    const bar = myActiveBar;
    const list = (myHotbars && myHotbars[bar]) ? myHotbars[bar] : [];
    const cap = petalCapDisplay(bar);
    for (let i = 0; i < cap; i++) {
        const s = document.createElement('div');
        // Общий вид слота как в меню: картинка + имя + счётчик.
        s.className = 'inv-slot hotbar-slot';
        s.dataset.bar = String(bar);
        s.dataset.slot = String(i);
        const it = (i < list.length) ? list[i] : null;
        if (!it || !it.id) {
            s.classList.add('empty-slot');
        } else {
            const col = rarityColors[Math.min(it.rarity || 0, rarityColors.length - 1)];
            s.title = '';
            attachPetalTip(s, it);
            const shape = document.createElement('div');
            shape.className = 'petal-shape';
            shape.innerHTML = petalSVG(it.kind, col);
            s.appendChild(shape);
            const nm = document.createElement('div');
            nm.className = 'inv-name';
            nm.textContent = it.name || kindName(it.kind);
            s.appendChild(nm);
            // Кулдаун: полупрозрачная «вуаль» сверху картинки вниз (показывает,
            // сколько ещё перезаряжается петаль — заполняется по мере кулдауна).
            const veil = document.createElement('div');
            veil.className = 'game-hotbar-cooldown';
            veil.style.height = '0%';
            s.appendChild(veil);
            gameHotbarVeils[i] = veil;
        }
        // Drop-цель: принимаем петаль и меняем местами (move).
        s.addEventListener('dragover', (ev) => { ev.preventDefault(); ev.dataTransfer.dropEffect = 'move'; });
        s.addEventListener('drop', (ev) => {
            ev.preventDefault();
            ev.stopPropagation();
            let data;
            try { data = JSON.parse(ev.dataTransfer.getData('text/plain')); }
            catch (e) { data = null; }
            if (!data) return;
            if (data.src === 'hot') hotDropAccepted = true;
            const from = (data.src === 'hot')
                ? { src: 'hot', bar: data.bar, slot: data.slot }
                : { src: 'inv', kind: data.kind, rarity: data.rarity };
            sendMsg('move', { from: from, toBar: bar, toSlot: i });
        });
        // Сам слот тащится в другой (обмен).
        s.setAttribute('draggable', String(!!(it && it.id)));
        s.addEventListener('dragstart', (ev) => {
            if (!it) { ev.preventDefault(); return; }
            ev.stopPropagation();
            hotDropAccepted = false;
            ev.dataTransfer.setData('text/plain', JSON.stringify({ src: 'hot', bar: bar, slot: i, kind: it.kind || '', rarity: it.rarity || 0 }));
            ev.dataTransfer.effectAllowed = 'move';
            s.classList.add('dragging');
        });
        s.addEventListener('dragend', () => {
            s.classList.remove('dragging');
            // В игре «куда угодно кроме хотбара» = тоже снять в инвентарь.
            if (it && it.id && !hotDropAccepted) unequipPetal(bar, i);
        });
        wrap.appendChild(s);
    }
    if (label) label.textContent = `Hotbar ${bar + 1}`;
}

// Открывает/закрывает панель инвентаря в меню.
function toggleInvPanel() {
    const panel = document.getElementById('invPanel');
    if (!panel) return;
    panel.classList.toggle('open');
    if (panel.classList.contains('open')) renderMenuInventory();
}

// Поиск в инвентаре: фильтрует стак по названию вида/редкости.
let invSearchQuery = '';
(function initInvSearch() {
    const inp = document.getElementById('invSearch');
    if (!inp) return;
    inp.addEventListener('input', () => {
        invSearchQuery = inp.value.trim();
        renderMenuInventory();
    });
    inp.addEventListener('keydown', (e) => e.stopPropagation());
})();

// Панель инвентаря принимает «сброс» надетого петаля — снимает его в запас.
(function initInvPanelDrop() {
    const panel = document.getElementById('invPanel');
    if (!panel) return;
    panel.addEventListener('dragover', (ev) => { ev.preventDefault(); ev.dataTransfer.dropEffect = 'move'; });
    panel.addEventListener('drop', (ev) => {
        ev.preventDefault();
        let data;
        try { data = JSON.parse(ev.dataTransfer.getData('text/plain')); }
        catch (e) { data = null; }
        if (!data) return;
        if (data.src === 'hot') {
            hotDropAccepted = true;
            unequipPetal(data.bar, data.slot);
        }
    });
})();

// Затемнение/осветление hex-строки цвета (для SVG-деталей петалей).
function shadeHex(cc, f) {
    if (!cc || cc[0] !== '#') return cc || '#ecf0f1';
    const n = parseInt(cc.slice(1), 16);
    const r = (n >> 16) & 255, g = (n >> 8) & 255, b = n & 255;
    const ch = v => Math.max(0, Math.min(255, Math.round(v * f)));
    return '#' + (((ch(r) << 16) | (ch(g) << 8) | ch(b))).toString(16).padStart(6, '0');
}

// SVG-изображение петеля по виду и цвету редкости. Уникальный силуэт на каждый вид.
function petalSVG(kind, color) {
    const c = color || '#ecf0f1';
    const dk = shadeHex(c, 0.62);
    const lt = shadeHex(c, 1.35);
    const polyPt = (n, r) => {
        let s = '';
        for (let i = 0; i < n; i++) {
            const a = -Math.PI / 2 + (i / n) * Math.PI * 2;
            s += `${(Math.cos(a) * r).toFixed(2)},${(Math.sin(a) * r).toFixed(2)} `;
        }
        return s.trim();
    };
    switch (kind) {
        case 'rhombus': {
            return `<svg viewBox="-14 -14 28 28" width="100%" height="100%">
                <polygon points="0,-12 9,0 0,12 -9,0" fill="${c}" stroke="#fff" stroke-width="1.2"/>
                <polygon points="0,-12 -9,0 0,12" fill="${dk}" opacity="0.35"/>
                <line x1="0" y1="-12" x2="0" y2="12" stroke="${lt}" stroke-width="1"/>
                <circle cx="0" cy="0" r="2.4" fill="${lt}" opacity="0.9"/>
            </svg>`;
        }
        case 'twirl': {
            // Вертушка: вытянутый ромб-дротик с наконечниками в обе стороны.
            return `<svg viewBox="-14 -14 28 28" width="100%" height="100%">
                <polygon points="0,-13.5 5.5,0 0,13.5 -5.5,0" fill="${c}" stroke="#fff" stroke-width="1.2"/>
                <polygon points="0,-13.5 -5.5,0 0,0" fill="${dk}" opacity="0.35"/>
                <line x1="0" y1="-12" x2="0" y2="12" stroke="${lt}" stroke-width="1" opacity="0.8"/>
                <circle cx="0" cy="0" r="2" fill="${lt}" opacity="0.9"/>
            </svg>`;
        }
        case 'rock': {
            return `<svg viewBox="-14 -14 28 28" width="100%" height="100%">
                <rect x="-7" y="-7" width="14" height="14" rx="3" fill="${c}" stroke="#fff" stroke-width="1.2"/>
                <rect x="-7.2" y="-8.5" width="12" height="3.4" rx="2" fill="${lt}" opacity="0.3"/>
                <rect x="-7.2" y="1.8" width="14.4" height="7" rx="2" fill="${dk}" opacity="0.4"/>
            </svg>`;
        }
        case 'pebble': {
            return `<svg viewBox="-14 -14 28 28" width="100%" height="100%">
                <ellipse cx="0" cy="0" rx="10" ry="7.2" fill="${c}" stroke="#fff" stroke-width="1.2"/>
                <ellipse cx="-2.5" cy="-2.8" rx="5" ry="2.6" fill="${lt}" opacity="0.35"/>
                <ellipse cx="1.5" cy="2.5" rx="4.2" ry="2.2" fill="${dk}" opacity="0.5"/>
            </svg>`;
        }
        case 'ball':
        case 'circle': {
            return `<svg viewBox="-14 -14 28 28" width="100%" height="100%">
                <circle cx="0" cy="0" r="10" fill="${c}" stroke="#fff" stroke-width="1.2"/>
                <circle cx="-3.5" cy="-3.5" r="3" fill="${lt}" opacity="0.55"/>
            </svg>`;
        }
        case 'orb': {
            return `<svg viewBox="-14 -14 28 28" width="100%" height="100%">
                <circle cx="0" cy="0" r="10" fill="${c}" stroke="#fff" stroke-width="1.2"/>
                <circle cx="-3.2" cy="-3.2" r="3.6" fill="${lt}" opacity="0.9"/>
                <ellipse cx="0" cy="5.5" rx="6" ry="2.8" fill="${dk}" opacity="0.4"/>
            </svg>`;
        }
        case 'hiveling': {
            return `<svg viewBox="-14 -14 28 28" width="100%" height="100%">
                <ellipse cx="0" cy="0" rx="10" ry="6.8" fill="${c}" stroke="#fff" stroke-width="1.2"/>
                <ellipse cx="-2.8" cy="0" rx="1.3" ry="5.8" fill="${dk}" opacity="0.85"/>
                <ellipse cx="2.4" cy="0" rx="1.3" ry="5.8" fill="${dk}" opacity="0.85"/>
                <ellipse cx="-3.8" cy="-6.2" rx="3.4" ry="2.2" fill="${lt}" opacity="0.7"/>
                <ellipse cx="4" cy="-6.2" rx="3.4" ry="2.2" fill="${lt}" opacity="0.7"/>
                <circle cx="6.2" cy="-1.8" r="1.3" fill="${lt}"/>
            </svg>`;
        }
        case 'void': {
            return `<svg viewBox="-14 -14 28 28" width="100%" height="100%">
                <circle cx="0" cy="0" r="10" fill="${dk}" stroke="#fff" stroke-width="1.2"/>
                <circle cx="0" cy="0" r="6.2" fill="none" stroke="${lt}" stroke-width="1.6" opacity="0.85"/>
                <circle cx="0" cy="0" r="8" fill="none" stroke="${lt}" stroke-width="1" opacity="0.45"/>
                <circle cx="0" cy="0" r="4.2" fill="#000" opacity="0.9"/>
            </svg>`;
        }
        case 'thorns':
        case 'fractalite': {
            if (kind === 'thorns') {
                return `<svg viewBox="-14 -14 28 28" width="100%" height="100%">
                    <polygon points="0,-13 10.5,9.5 -10.5,9.5" fill="${c}" stroke="#fff" stroke-width="1.2"/>
                    <path d="M-0.5,-8.5 L3,3.5" stroke="${lt}" stroke-width="1" opacity="0.55"/>
                </svg>`;
            }
            return `<svg viewBox="-14 -14 28 28" width="100%" height="100%">
                <polygon points="0,-12 11,9 -11,9" fill="${c}" stroke="#fff" stroke-width="1.2"/>
                <polygon points="0,-1 5,5.5 -5,5.5" fill="${dk}" opacity="0.95"/>
                <path d="M0,-10 L4.5,3.5" stroke="${lt}" stroke-width="1" opacity="0.5"/>
            </svg>`;
        }
        case 'shard': {
            return `<svg viewBox="-14 -14 28 28" width="100%" height="100%">
                <polygon points="0,-13 1.6,-2.5 1.2,5.5 4.2,2 5,9 0.5,7.5 -4.2,10 -3,1" fill="${c}" stroke="#fff" stroke-width="1.2"/>
                <path d="M-0.5,-10 L-1.2,-2.5 L0.3,3.5" stroke="${dk}" stroke-width="1" fill="none" opacity="0.9"/>
            </svg>`;
        }
        case 'blade': {
            return `<svg viewBox="-14 -14 28 28" width="100%" height="100%">
                <polygon points="-14,0 10,-3.8 3,-1.6 3,1.6 10,3.8" fill="${c}" stroke="#fff" stroke-width="1.2"/>
                <rect x="5" y="-2.4" width="8" height="4.8" rx="2.5" fill="${dk}"/>
                <path d="M-9,0 L2.5,0" stroke="${dk}" stroke-width="1" opacity="0.6"/>
            </svg>`;
        }
        case 'viper': {
            return `<svg viewBox="-14 -14 28 28" width="100%" height="100%">
                <path d="M-6,8.5 Q3.5,4.5 3.5,-8 L-0.5,-8.5 Q-1.2,2 -8.5,5.5 Z" fill="${c}" stroke="#fff" stroke-width="1.2"/>
                <path d="M-6.8,6.2 Q0.2,4 1.2,-5 L2.6,-4.4 Q1,3.3 -5.2,5.5 Z" fill="${lt}" opacity="0.5"/>
            </svg>`;
        }
        case 'cube': {
            return `<svg viewBox="-14 -14 28 28" width="100%" height="100%">
                <polygon points="-8,7 8,7 8,-7 -8,-7" fill="${c}" stroke="#fff" stroke-width="1.2"/>
                <polygon points="-8,-7 8,-7 11,-10 -5,-10" fill="${lt}" opacity="0.7"/>
                <polygon points="8,-7 11,-10 11,4 8,7" fill="${dk}" opacity="0.5"/>
                <path d="M-8,-7 L-5,-10 M8,-7 L11,-10 M8,7 L11,4" stroke="#fff" stroke-width="0.8" opacity="0.6" fill="none"/>
            </svg>`;
        }
        case 'bulwark': {
            return `<svg viewBox="-14 -14 28 28" width="100%" height="100%">
                <path d="M-8.5,4.5 H8.5 V-4.5 H7.5 V-9 H4.5 V-4.5 H1.5 V-9 H-1.5 V-4.5 H-4.5 V-9 H-7.5 V-4.5 H-8.5 Z" fill="${c}" stroke="#fff" stroke-width="1.5"/>
                <rect x="-1.3" y="0.2" width="2.6" height="4.3" rx="2" fill="${dk}" opacity="0.85"/>
                <rect x="-1.3" y="-8" width="2.6" height="3" rx="1" fill="${lt}" opacity="0.35"/>
            </svg>`;
        }
        case 'penta':
        case 'hex': {
            if (kind === 'hex') {
                // Плосковершинный шестиугольник-соты: ровный контур + затемнённая ячейка.
                const hx = (r) => {
                    let s = '';
                    for (let i = 0; i < 6; i++) {
                        const a = Math.PI / 6 + (i / 6) * Math.PI * 2;
                        s += `${(Math.cos(a) * r).toFixed(2)},${(Math.sin(a) * r).toFixed(2)} `;
                    }
                    return s.trim();
                };
                return `<svg viewBox="-14 -14 28 28" width="100%" height="100%">
                    <polygon points="${hx(10.5)}" fill="${c}" stroke="#fff" stroke-width="1.2"/>
                    <polygon points="${hx(5.5)}" fill="${dk}" opacity="0.85"/>
                    <polygon points="${hx(5.5)}" fill="none" stroke="${lt}" stroke-width="1" opacity="0.75"/>
                </svg>`;
            }
            return `<svg viewBox="-14 -14 28 28" width="100%" height="100%">
                <polygon points="${polyPt(5, 10.5)}" fill="${c}" stroke="#fff" stroke-width="1.2"/>
                <polygon points="${polyPt(5, 5.5)}" fill="none" stroke="${lt}" stroke-width="1" opacity="0.7"/>
            </svg>`;
        }
        case 'shield': {
            return `<svg viewBox="-14 -14 28 28" width="100%" height="100%">
                <path d="M-8,-9 H8 V2 L0,12 -8,2 Z" fill="${c}" stroke="#fff" stroke-width="1.2"/>
                <rect x="-7.2" y="-6.6" width="14.4" height="1.6" rx="1" fill="${dk}" opacity="0.5"/>
                <circle cx="0" cy="-0.2" r="2.6" fill="${lt}" opacity="0.55"/>
            </svg>`;
        }
        case 'cog': {
            let pts = '';
            for (let i = 0; i < 16; i++) {
                const a = (i / 16) * Math.PI * 2;
                const r = (i % 2 === 0) ? 8 : 5.2;
                pts += `${(Math.cos(a) * r).toFixed(2)},${(Math.sin(a) * r).toFixed(2)} `;
            }
            return `<svg viewBox="-14 -14 28 28" width="100%" height="100%">
                <polygon points="${pts.trim()}" fill="${c}" stroke="#fff" stroke-width="1.2"/>
                <path d="M4.2,0 A4.2,4.2 0 1 1 -4.2,0 A4.2,4.2 0 1 1 4.2,0 M2,0 A2,2 0 1 0 -2,0 A2,2 0 1 0 2,0" fill-rule="evenodd" fill="${c}"/>
                <circle cx="1.3" cy="0" r="0.9" fill="${lt}" opacity="0.7"/>
            </svg>`;
        }
        case 'whirl': {
            // Вихрь: круглый спиннер — три лопасти вокруг тела с тёмной осью.
            let lobes = '';
            for (let i = 0; i < 3; i++) {
                const a = (i / 3) * Math.PI * 2 - Math.PI / 2;
                lobes += `<circle cx="${(Math.cos(a) * 7.6).toFixed(2)}" cy="${(Math.sin(a) * 7.6).toFixed(2)}" r="4.4" fill="${c}"/>`;
            }
            return `<svg viewBox="-14 -14 28 28" width="100%" height="100%">
                ${lobes}
                <circle cx="0" cy="0" r="6.2" fill="${c}" stroke="#fff" stroke-width="1.2"/>
                <circle cx="0" cy="0" r="2.2" fill="${dk}"/>
                <circle cx="0" cy="0" r="0.8" fill="${lt}"/>
            </svg>`;
        }
        case 'spark': {
            return `<svg viewBox="-14 -14 28 28" width="100%" height="100%">
                <path d="M0,-11 L2.6,-2.6 L11,0 L2.6,2.6 L0,11 L-2.6,2.6 L-11,0 L-2.6,-2.6 Z" fill="${c}" stroke="#fff" stroke-width="1.2"/>
                <circle cx="0" cy="0" r="2.4" fill="${lt}" opacity="0.85"/>
            </svg>`;
        }
        case 'fragwall': {
            const bricks = (x, y, n) => {
                let r = '';
                for (let i = 0; i < n; i++) {
                    r += `<rect x="${(x + i * 5.6).toFixed(1)}" y="${y}" width="4.4" height="4.2" rx="1" fill="${c}" stroke="#fff" stroke-width="0.8"/>`;
                }
                return r;
            };
            return `<svg viewBox="-14 -14 28 28" width="100%" height="100%">
                <rect x="-8" y="-5.5" width="16" height="11" rx="2" fill="${dk}" stroke="#fff" stroke-width="1.2"/>
                ${bricks(-7.8, -5, 3)}
                ${bricks(-5, 0.3, 2)}
                <path d="M-7,4.5 L-2,0.2 L1.2,-3.8 M-1.5,4.2 L2.6,1.2" stroke="${dk}" stroke-width="1.4" fill="none" opacity="0.9"/>
            </svg>`;
        }
        case 'entropy': {
            let sp = '', dots = '';
            for (let i = 0; i < 6; i++) {
                const a = (i / 6) * Math.PI * 2;
                const x1 = (Math.cos(a) * 5.8).toFixed(1), y1 = (Math.sin(a) * 5.8).toFixed(1);
                const x2 = (Math.cos(a) * 9.8).toFixed(1), y2 = (Math.sin(a) * 9.8).toFixed(1);
                sp += `<path d="M${x1},${y1} L${x2},${y2}" stroke="${c}" stroke-width="2.2"/>`;
                dots += `<circle cx="${x2}" cy="${y2}" r="1.3" fill="${c}"/>`;
            }
            return `<svg viewBox="-14 -14 28 28" width="100%" height="100%">
                <circle cx="0" cy="0" r="5.5" fill="${dk}"/>
                ${sp}
                ${dots}
                <circle cx="0" cy="0" r="1.7" fill="${lt}"/>
            </svg>`;
        }
        case 'singularity': {
            // Singularity: тёмное ядро с вихревыми спицами (как чёрная дыра).
            let sp = '', dots = '';
            for (let i = 0; i < 6; i++) {
                const a = (i / 6) * Math.PI * 2;
                const x1 = (Math.cos(a) * 4.5).toFixed(1), y1 = (Math.sin(a) * 4.5).toFixed(1);
                const x2 = (Math.cos(a) * 10.5).toFixed(1), y2 = (Math.sin(a) * 10.5).toFixed(1);
                sp += `<path d="M${x1},${y1} L${x2},${y2}" stroke="${dk}" stroke-width="2.5"/>`;
                dots += `<circle cx="${x2}" cy="${y2}" r="1.5" fill="${dk}"/>`;
            }
            return `<svg viewBox="-14 -14 28 28" width="100%" height="100%">
                <circle cx="0" cy="0" r="5" fill="#0a0618"/>
                ${sp}
                ${dots}
                <circle cx="0" cy="0" r="2" fill="${lt}"/>
            </svg>`;
        }
        default: {
            // Oval (и все фолбэки).
            return `<svg viewBox="-14 -14 28 28" width="100%" height="100%">
                <path d="M0,-11 C9,0 9,6 0,11 C-9,6 -9,0 0,-11Z" fill="${c}" stroke="#fff" stroke-width="1.2"/>
                <path d="M0,-9 Q1.5,0 -1,8.5" stroke="${dk}" stroke-width="1" opacity="0.5" fill="none"/>
            </svg>`;
        }
        case 'tear': {
            return `<svg viewBox="-14 -14 28 28" width="100%" height="100%">
                <path d="M0,-14 L8.5,-1 Q7,11 0,9.8 Q-7,11 -8.5,-1 Z" fill="${c}" stroke="#fff" stroke-width="1.2"/>
                <ellipse cx="-3.4" cy="-5.6" rx="2.4" ry="3.8" fill="${lt}" opacity="0.55"/>
            </svg>`;
        }
        case 'line': {
            return `<svg viewBox="-14 -14 28 28" width="100%" height="100%">
                <path d="M0,9 L0,-12" stroke="#fff" stroke-width="1.4" fill="none"/>
                <circle cx="0" cy="-13.4" r="3.8" fill="${c}" stroke="#fff" stroke-width="1.2"/>
                <circle cx="-1.3" cy="-14.2" r="1.4" fill="${lt}" opacity="0.7"/>
                <path d="M-4,9 L4,9" stroke="${dk}" stroke-width="1.6"/>
            </svg>`;
        }
        case 'compass': {
            let rose = '';
            for (let i = 0; i < 4; i++) {
                const a = i * Math.PI / 2;
                rose += `<circle cx="${(Math.cos(a) * 4.6).toFixed(2)}" cy="${(Math.sin(a) * 4.6).toFixed(2)}" r="1" fill="${lt}"/>`;
            }
            return `<svg viewBox="-14 -14 28 28" width="100%" height="100%">
                <circle cx="0" cy="0" r="11" fill="${c}" stroke="#fff" stroke-width="1.2"/>
                <circle cx="0" cy="0" r="6.6" fill="${dk}" opacity="0.9"/>
                ${rose}
                <path d="M-3.4,0 L3.4,0" stroke="${lt}" stroke-width="1" opacity="0.5"/>
                <path d="M-3.6,0 L4,0 M4,0 L2.2,1.6 M4,0 L2.2,-1.6" stroke="#ff3b3b" stroke-width="1.5" fill="none" stroke-linecap="round"/>
            </svg>`;
        }
        case 'saturn': {
            return `<svg viewBox="-14 -14 28 28" width="100%" height="100%">
                <ellipse cx="0" cy="-0.6" rx="13" ry="4.6" fill="${dk}" opacity="0.85"/>
                <circle cx="0" cy="0.4" r="8" fill="${c}" stroke="#fff" stroke-width="1.2"/>
                <ellipse cx="-1.9" cy="-2.2" rx="3.2" ry="1.9" fill="${lt}" opacity="0.5"/>
                <ellipse cx="0" cy="5.4" rx="4.8" ry="1.9" fill="${dk}" opacity="0.45"/>
                <path d="M-12.8,-1.8 Q0,4 12.8,-1.8" stroke="${lt}" stroke-width="1.4" fill="none"/>
            </svg>`;
        }
        case 'rune': {
            const hx6 = (r) => {
                let s = '';
                for (let i = 0; i < 6; i++) {
                    const a = Math.PI / 6 + (i / 6) * Math.PI * 2;
                    s += `${(Math.cos(a) * r).toFixed(2)},${(Math.sin(a) * r).toFixed(2)} `;
                }
                return s.trim();
            };
            return `<svg viewBox="-14 -14 28 28" width="100%" height="100%">
                <polygon points="${hx6(11)}" fill="${c}" stroke="#fff" stroke-width="1.2"/>
                <polygon points="${hx6(6.6)}" fill="${dk}" opacity="0.9"/>
                <path d="M-2,8 L2,8 L2,-6 M2,-6 L-2.4,-1.8 M-2.4,-1.8 L-0.6,8" stroke="${lt}" stroke-width="1.7" fill="none" stroke-linecap="round"/>
            </svg>`;
        }
        case 'eye': {
            return `<svg viewBox="-14 -14 28 28" width="100%" height="100%">
                <path d="M0,-10 Q13,0 0,10 Q-13,0 0,-10Z" fill="${c}" stroke="#fff" stroke-width="1.2"/>
                <ellipse cx="0" cy="0" rx="6.4" ry="5" fill="#fff"/>
                <circle cx="0" cy="0" r="3.4" fill="${c}"/>
                <circle cx="0" cy="0" r="1.6" fill="#000"/>
                <circle cx="-1.4" cy="-1.6" r="0.9" fill="#fff" opacity="0.9"/>
            </svg>`;
        }
        case 'spike': {
            // Шипастое кольцо: ободок с острыми шипами наружу (обводка вокруг игрока).
            let pts = '';
            const N = 10;
            for (let i = 0; i < N * 2; i++) {
                const a = (i / (N * 2)) * Math.PI * 2;
                const r = (i % 2 === 0) ? 11.6 : 5.6;
                pts += `${(Math.cos(a) * r).toFixed(2)},${(Math.sin(a) * r).toFixed(2)} `;
            }
            return `<svg viewBox="-14 -14 28 28" width="100%" height="100%">
                <polygon points="${pts.trim()}" fill="${c}" stroke="#fff" stroke-width="1.2"/>
                <circle cx="0" cy="0" r="2.8" fill="${dk}" opacity="0.9"/>
                <circle cx="0" cy="0" r="1.2" fill="${lt}"/>
            </svg>`;
        }
    }
}

// Имя редкости — из единого реестра js/rarities.js (16 редкостей).

// HEX-цвет текста по индексу редкости (для подписи красочной подписи).
function rarityHex(ri) {
    return rarityColors[Math.max(0, Math.min(ri || 0, rarityColors.length - 1))];
}

// rarityNum — то же, но числом 0xRRGGBB (для PIXI.Graphics.beginFill).
function rarityNum(ri) {
    return parseInt(rarityHex(ri).slice(1), 16);
}

async function apiCall(endpoint, body) {
    const res = await fetch(endpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
    });
    return res.json();
}

// Загружает профиль игрока (уровень/XP) с сервера по токену.
// Используется при перезагрузке страницы, чтобы меню показало актуальные данные.
async function loadProfile() {
    if (!authToken) return;
    try {
        const data = await apiCall('/api/profile', { token: authToken });
        if (data.ok) {
            myNickname = data.nick;
            ensureSpecialColor();
            myLevel = data.level;
            myXP = data.xp;
            myXPMax = data.xpMax;
            // Загружаем сохранённый запас и раскладку хотбаров.
            if (Array.isArray(data.inventory)) myInventory = data.inventory;
            if (Array.isArray(data.hotbars) && data.hotbars.length === 2) myHotbars = data.hotbars;
            if (data.activeBar === 0 || data.activeBar === 1) myActiveBar = data.activeBar;
if (data.pending) craftPending = data.pending;
            if (typeof data.bestWave === 'number') {
                const bw = document.getElementById('menuBestWave');
                if (bw) bw.textContent = data.bestWave;
            }
            updateMenuLevel();
            renderGameHotbar();
            updateCraftBtn();
        } else if (data.error && data.error.includes('token')) {
            authToken = '';
            localStorage.removeItem('authToken');
            showScreen('login');
        }
    } catch (e) {
        console.error('PROFILE fetch error:', e);
    }
}

// ----- Правки раскладки В МЕНЮ: напрямую через REST (без WS/мира) -----
function menuOpen() { return menu.classList.contains('active'); }

function applyProfile(data) {
    if (!data || !data.ok) return;
    if (data.nick) myNickname = data.nick;
    ensureSpecialColor();
    if (Array.isArray(data.inventory)) myInventory = data.inventory;
    if (Array.isArray(data.hotbars) && data.hotbars.length === 2) myHotbars = data.hotbars;
    if (data.activeBar === 0 || data.activeBar === 1) myActiveBar = data.activeBar;
    if (data.pending) craftPendingPetal = data.pending;
    myLevel = data.level; myXP = data.xp; myXPMax = data.xpMax;
    lastInvSig = ''; lastHotbarSig = '';
    updateMenuLevel();
    renderGameHotbar();
    updateCraftBtn();
}

async function menuApi(op, payload) {
    if (!authToken) return;
    try {
        const data = await apiCall('/api/menu-' + op, Object.assign({ token: authToken }, payload));
        applyProfile(data);
    } catch (e) {
        console.error('MENU API ' + op + ':', e);
    }
}

// Универсальные действия: если открыто меню — REST в БД, иначе — WS (в игре).
function movePetal(from, toBar, toSlot) {
    if (menuOpen()) menuApi('move', { from: from, toBar: toBar, toSlot: toSlot });
    else sendMsg('move', { from: from, toBar: toBar, toSlot: toSlot });
}
// Отметка, что последний drag из хотбара был успешно брошен В ВАЛИДНУЮ цель
// (слот хотбара или панель инвентаря). Нужна для fallback: если перетаскиваемый
// петаль бросили куда угодно кроме хотбара/инвентаря — снять его в инвентарь.
let hotDropAccepted = false;
// Guard «шифт-хувера»: не снимать повторно один и тот же слот, пока над ним
// держат мышь с зажатым Shift (срабатывание один раз на наведение).
let shiftUnequipArmed = null; // id слота, над которым уже сработал шифт
function unequipPetal(bar, slot) {
    if (menuOpen()) menuApi('unequip', { bar: bar, slot: slot });
    else sendMsg('unequip', { bar: bar, slot: slot });
}
function dropPetal(from) {
    if (menuOpen()) {
        menuApi('drop', { from: from });
    } else if (from && from.src === 'hot') {
        sendMsg('dropHotbar', { bar: from.bar, slot: from.slot });
    } else {
        sendMsg('drop', { kind: from && from.kind, rarity: from && from.rarity });
    }
}
function setHotbar(bar) {
    if (menuOpen()) { menuApi('setbar', { bar: bar }); myActiveBar = bar; }
    else sendMsg('setbar', { bar: bar });
}

// ----- Крафт (катушка) в меню -----
// В катушку вкладываются петали одного вида/редкости. Обычный клик кладёт 5,
// Shift — все (до 1-4 остатка). Крутка запускает катушку по ~5 секунд на каждую
// пятёрку, ускоренно раскручивая и показывая в центре полученный петаль и его
// количество; так повторяется, пока в катушке не останется 1-4 петали.
let craftPileCount = 0;      // сколько петалей сейчас вложено (зарезервировано)
let craftPileKind = null;    // вид вложенных петалей
let craftPileRarity = 0;     // редкость вложенных петалей
let craftPending = [];       // результаты удачного крафта, ждут «Забрать»
let craftSpinning = false;   // идёт анимация крутки (заблокировано)

// Шанс успеха крафта — из единого реестра js/rarities.js (CRAFT_CHANCE_PCT,
// синхронизирован с сервером; раньше локальная копия разошлась с ним).

// Текущий стек катушки — ключ "kind|rarity".
function pileKey(kind, rarity) { return (kind || '') + '|' + (rarity || 0); }
const craftPileKey = () => pileKey(craftPileKind, craftPileRarity);

// Сколько петалей этого стека уже вложено в катушку (для уменьшения в списке).
function pileReservedFor(kind, rarity) {
    return (craftPileCount > 0 && craftPileKind === kind && craftPileRarity === (rarity || 0))
        ? craftPileCount : 0;
}

// Стаки в инвентаре, которые можно крафтить (за вычетом уже вложенных). Человек
// может докладывать по 5 (или все с Shift), пока не упрётся в доступное.
function craftChoices() {
    const out = [];
    for (const it of (myInventory || [])) {
        const raw = Number(it.count) || 1;
        const kind = it.kind || '';
        const rarity = it.rarity || 0;
        const avail = raw - pileReservedFor(kind, rarity);
        out.push({ kind, rarity, avail, total: raw, name: it.name || kindName(kind) });
    }
    return out;
}

function setCraftResult(msg) {
    const box = document.getElementById('craftResult');
    if (box) box.textContent = msg;
}

const waitMs = ms => new Promise(r => setTimeout(r, ms));

// Рисует петали по кругу в полосе катушки (максимум 5 — один запуск).
function drawWheelBand(kind, rarity, count) {
    const band = document.getElementById('craftWheelBand');
    if (!band) return;
    band.innerHTML = '';
    if (!kind || count <= 0) return;
    const col = rarityColors[Math.min(rarity || 0, rarityColors.length - 1)];
    const show = Math.min(count, 5);
    const R = 60;
    for (let i = 0; i < show; i++) {
        const a = (i / show) * Math.PI * 2 - Math.PI / 2;
        const orb = document.createElement('div');
        orb.className = 'craft-petal-orb';
        orb.innerHTML = petalSVG(kind, col);
        orb.style.left = (95 + Math.cos(a) * R) + 'px';
        orb.style.top = (95 + Math.sin(a) * R) + 'px';
        band.appendChild(orb);
    }
}

function renderSpinner() {
    drawWheelBand(craftPileKind, craftPileRarity, craftPileCount);
}

// Центр катушки: петаль результата + количество (или «пусто»).
function renderCenter(kind, rarity, count, pop) {
    const icon = document.getElementById('craftCenterIcon');
    const cnt = document.getElementById('craftCenterCount');
    if (!icon || !cnt) return;
    if (!kind) {
        icon.innerHTML = '';
        cnt.textContent = '';
        return;
    }
    const col = rarityColors[Math.min(rarity, rarityColors.length - 1)];
    icon.innerHTML = petalSVG(kind, col);
    cnt.textContent = '×' + (count || 1);
    if (pop) {
        // Успех: петаль в центре становится ЧУТЬ БОЛЬШЕ (пока ждёт «Забрать»)
        // с золотым свечением, количество тоже крупнее.
        icon.classList.add('won');
        cnt.classList.add('won');
        icon.classList.remove('pop');
        void icon.offsetWidth;
        icon.classList.add('pop');
    } else {
        icon.classList.remove('won');
        cnt.classList.remove('won');
    }
}

function renderCraftPanel() {
    const panel = document.getElementById('craftPanel');
    const pileC = document.getElementById('craftPileCount');
    const pileI = document.getElementById('craftPileInfo');
    const chance = document.getElementById('craftChance');
    const bi = document.getElementById('craftBtnInner');
    const resetBtn = document.getElementById('craftResetBtn');
    const inv = document.getElementById('craftInventory');
    if (!panel || !panel.classList.contains('open')) return;

    // Кнопка сброса активна, пока в катушке что-то есть и нет незабранного результата.
    if (resetBtn) resetBtn.disabled = !(craftPileCount > 0 && craftPending.length === 0) || craftSpinning;

    // Катушка.
    renderSpinner();

    // Информация о вложенном.
    if (pileC) pileC.textContent = (craftPileCount > 0 || craftPending.length > 0)
        ? `Added: ${craftPileCount}` : 'Added: 0';
    if (pileI) {
        if (craftPileCount > 0) pileI.textContent = `${kindName(craftPileKind)} ${rarityName(craftPileRarity)}`;
        else if (craftPending.length > 0) pileI.textContent = 'Result is in the wheel';
        else pileI.textContent = '—';
    }

    // Шанс (без показа числа попыток — оно считается при крутках).
    if (chance) {
        if (craftPileRarity >= 7) {
            chance.textContent = 'Cannot craft from Chaos';
            chance.style.color = '#e74c3c';
        } else if (craftPileCount >= 5) {
            chance.textContent = `Success Chance = ${CRAFT_CHANCE_PCT[craftPileRarity]} %`;
            chance.style.color = '#ffd54f';
        } else {
            chance.textContent = craftPileCount > 0 ? 'Add more (x5)' : '—';
            chance.style.color = '#8a93a6';
        }
    }

    // Кнопка.
    if (bi) {
        bi.disabled = !!craftSpinning;
        if (craftPending.length > 0) {
            bi.textContent = `Take (${craftPending.length})`;
            bi.style.background = craftBtnGreen();
        } else if (craftPileCount >= 5 && craftPileRarity < 7) {
            bi.textContent = 'Spin';
            bi.style.background = craftBtnGold();
        } else {
            bi.textContent = craftPileRarity >= 7 ? 'Cannot craft further' : 'Need 5+ shapes';
            bi.disabled = true;
            bi.style.background = craftBtnGold();
        }
    }

    // Инвентарь крафта (число уменьшается по мере вкладывания).
    if (inv) {
        inv.innerHTML = '';
        const labels = { '': 'Click = +5 · Shift = all' };
        const choices = craftChoices().filter(c => c.avail > 0 || c.total > 0);
        if (choices.length === 0) {
            const e = document.createElement('div');
            e.className = 'inv-empty';
            e.textContent = 'No shapes in inventory for crafting';
            e.style.cssText = 'width:auto;height:40px;font-size:10px;color:#8a93a6;';
            inv.appendChild(e);
        }
        for (const it of choices) {
            const slot = document.createElement('div');
            const col = rarityColors[Math.min(it.rarity, rarityColors.length - 1)];
            slot.className = 'craft-slot';
            // Не крафтим из Chaos; пока идут результаты — только «Забрать».
            const disabled = it.rarity >= 7 || it.avail <= 0 || craftPending.length > 0 || craftSpinning;
            if (!disabled) {
                slot.classList.add('pickable');
                slot.title = `${it.name} ${rarityName(it.rarity)} (available ${it.avail})\nClick = +5 · Shift = all`;
            } else {
                slot.title = it.rarity >= 7 ? 'Cannot craft from Chaos'
                    : (it.avail <= 0 ? 'All shapes added to the wheel'
                        : (craftPending.length > 0 ? 'First «Take» the result' : ''));
                slot.style.opacity = '0.45';
            }
            const shape = document.createElement('div');
            shape.className = 'petal-shape';
            shape.innerHTML = petalSVG(it.kind, col);
            slot.appendChild(shape);
            const nm = document.createElement('div');
            nm.className = 'inv-name';
            nm.textContent = `${it.name} ${rarityName(it.rarity)}`;
            slot.appendChild(nm);
            const badge = document.createElement('div');
            badge.className = 'cat-count';
            badge.textContent = '×' + Math.max(0, it.avail);
            slot.appendChild(badge);
            if (!disabled) slot.addEventListener('click', e => craftAddToPile(it.kind, it.rarity, it.total, e.shiftKey));
            inv.appendChild(slot);
        }
    }
}

function craftBtnGold() { return 'linear-gradient(180deg, #ffe27a, #ffb300)'; }
function craftBtnGreen() { return 'linear-gradient(180deg, #5bd07a, #2e9e4f)'; }

// Добавить петали в катушку. shift=true кладёт ВСЕ доступные, иначе 5 (или остаток).
// Клик по петали другого вида/редкости автоматически сбрасывает текущую крутку
// (петали в инвентарь возвращаются — они ещё не потрачены на сервере).
function craftAddToPile(kind, rarity, total, shift) {
    if (craftSpinning) return;
    if (craftPending.length > 0) { setCraftResult('First «Take» the result.'); return; }
    rarity = rarity || 0;
    if (rarity >= 7) { setCraftResult('Cannot craft from Chaos.'); return; }

    // Другой вид/редкость => переключаемся на него, сбрасываем текущий.
    if (craftPileCount > 0 && (craftPileKind !== kind || craftPileRarity !== rarity)) {
        craftResetPile('Wheel switched to another petal.');
    }

    const avail = Number(total) - pileReservedFor(kind, rarity);
    if (avail <= 0) { setCraftResult('These shapes are already in the wheel.'); return; }
    if (craftPileCount === 0) {
        craftPileKind = kind;
        craftPileRarity = rarity;
    }
    const add = shift ? avail : Math.min(5, avail);
    craftPileCount += add;
    renderCraftPanel();
    const rem = craftPileCount % 5;
    if (rem === 0) {
        setCraftResult('');
    } else if (shift) {
        setCraftResult(`Added ${craftPileCount} — the wheel spins it all down to 1-4 left.`);
    } else {
        setCraftResult('');
    }
}

// Сброс катушки: резерв петалей отменяется и петали снова видны в инвентаре.
function craftResetPile(msg) {
    craftPileCount = 0;
    craftPileKind = null;
    craftPileRarity = 0;
    renderCenter(null, 0, 0);
    const band = document.getElementById('craftWheelBand');
    if (band) band.style.transform = 'rotate(0deg)';
    if (msg) setCraftResult(msg);
    renderCraftPanel();
}

// Вращение полосы с постоянным угловым ускорением: скорость растёт вплоть до
// конца (~5 с) — «ускорение к пятой секунде», затем резкая остановка с результатом.
async function spinOnce(band, ms) {
    return new Promise(res => {
        if (!band) { res(); return; }
        const T = ms;
        const ALPHA = (2 * 5400) / (T * T); // к концу ~5400° при постоянном ускорении
        const t0 = typeof performance !== 'undefined' ? performance.now() : Date.now();
        const step = now => {
            const t = now - t0;
            if (t >= T) { band.style.transform = `rotate(${0.5 * ALPHA * T * T}deg)`; res(); return; }
            band.style.transform = `rotate(${0.5 * ALPHA * t * t}deg)`;
            requestAnimationFrame(step);
        };
        requestAnimationFrame(step);
    });
}

// Перезапуск CSS-анимации вспышки (успех/провал).
function craftFlash(el, cls) {
    if (!el) return;
    el.classList.remove('flash-ok', 'flash-fail');
    void el.offsetWidth;
    el.classList.add(cls);
}

// Крутка: сервер «сливает» пачку петалей, перековывая по 5, пока не останется
// 1-4 (или 0), и возвращает итоги. Клиент запускает ОДНУ анимацию катушки
// (~5 с, ускорение), затем показывает суммарный результат: успешные + потеряно.
// После крутки полоса всегда пустая (петали вернулись в инвентарь), в центре
// — полученный петаль, если успех был.
async function doCraft() {
    if (craftSpinning || craftPending.length > 0) return;
    const count = craftPileCount;
    if (count < 5 || craftPileRarity >= 7) return;
    const k = craftPileKind, r = craftPileRarity; // текущий вид/редкость крутки
    craftSpinning = true;
    renderCraftPanel();
    const band = document.getElementById('craftWheelBand');
    const inner = document.getElementById('craftSpinnerInner');

    // Итоги (внешний охват, чтобы применить их после перерисовки).
    let outKind = null, outRar = 0, outCount = 0, didWin = false, outRem = 0;

    try {
        // Внутри игры — WS-крафт по ЖИВОМУ инвентарю (меню — HTTP к БД).
        let data;
        if (menuOpen()) {
            data = await apiCall('/api/craft', { token: authToken, kind: k, rarity: r, count: count });
        } else {
            data = await requestLive('craftLive', { kind: k, rarity: r, count: count });
            if (data && data.error === 'Need5') {
                craftSpinning = false;
                renderCraftPanel();
                setCraftResult('Need 5+ shapes');
                return;
            }
        }
        if (!data || !data.ok) {
            craftSpinning = false;
            renderCraftPanel();
            renderCenter(null, 0, 0);
            setCraftResult(data.error || 'Craft error');
            return;
        }
        // Сервер уже всё списал; берём его инвентарь как истину.
        if (Array.isArray(data.inventory)) { myInventory = data.inventory; lastInvSig = ''; }
        renderMenuInventory();
        // ОДНА раскрутка (ускорение к пятой секунде) — без «катушка №N».
        setCraftResult('Spinning…');
        if (band) band.style.transform = 'rotate(0deg)';
        await spinOnce(band, 5000);
        if (band) band.style.transform = 'rotate(0deg)';

        const successes = data.successes || 0;
        const lost = data.lost || 0;
        const nextRar = data.nextRarity || data.next || (r + 1);
        craftPending = data.pending || [];

        // Сколько петалей того же вида/редкости осталось в стопке (0..4 — «хвост»
        // при большом вложении; при малом — вся оставшаяся стопка).
        let remaining = 0;
        for (const it of (myInventory || [])) {
            if ((it.kind || '') === (k || '') && (it.rarity || 0) === r) { remaining = it.count || 0; break; }
        }
        outKind = data.kind; outRar = nextRar; outCount = successes;
        didWin = successes > 0; outRem = remaining;
        // Что реально сгорело в крафте (ушло из стопки навсегда): каждая успешная
        // попытка забрала 5, каждый провал — lost петалей. «count - remaining»
        // здесь НЕ годится: remaining — вся стопка, а не только вложенная часть.
        const consumed = lost + 5 * successes;

        // Катушка больше ничего не резервирует — петали вернулись в инвентарь.
        craftPileCount = 0;
        craftPileKind = null;
        craftPileRarity = 0;

        // После крафта полоса всегда пустая; в центре — результат, если он был.
        drawWheelBand(null, 0, 0);
        if (successes > 0) {
            renderCenter(data.kind, nextRar, successes, true);
            craftFlash(inner, 'flash-ok');
            if (craftPending.length > 0) {
                setCraftResult(`Successful: ${successes} · Lost: ${consumed}\u00A0— press «Take» to claim ${craftPending.length} shapes.`);
            } else {
                setCraftResult(`Successful: ${successes} · Lost: ${consumed}\u00A0— ${kindName(data.kind)} ${rarityName(nextRar)}`);
            }
        } else if (lost > 0) {
            craftFlash(inner, 'flash-fail');
            if (remaining > 0) {
                setCraftResult(`Failed — Lost: ${consumed} · Remaining: ${remaining} shapes.`);
            } else {
                setCraftResult(`Failed — Lost: ${consumed} shapes.`);
            }
        } else {
            craftFlash(inner, 'flash-ok');
            setCraftResult(`Lost: ${consumed} \u00A0· Remaining: ${remaining}`);
        }
    } catch (e) {
        if (band) band.style.transform = 'rotate(0deg)';
        craftSpinning = false;
        renderCraftPanel();
        renderCenter(null, 0, 0);
        setCraftResult('Connection error');
        return;
    }

    craftSpinning = false;
    renderCraftPanel();
    updateCraftBtn();
    // После перерисовки полоса остаётся пустой; в центре — только результат.
    drawWheelBand(null, 0, 0);
    if (didWin) renderCenter(outKind, outRar, outCount, true);
    else renderCenter(null, 0, 0);
}

// «Забрать» результаты крафта в инвентарь.
async function takeCraft() {
    try {
        let data;
        if (menuOpen()) {
            data = await apiCall('/api/craft-take', { token: authToken });
        } else {
            data = await requestLive('craftLiveTake', {});
        }
        if (data && data.ok) {
            if (Array.isArray(data.inventory)) { myInventory = data.inventory; lastInvSig = ''; }
            craftPending = [];
            renderCenter(null, 0, 0);
            setCraftResult('Shapes returned to inventory!');
            renderMenuInventory();
        } else {
            setCraftResult(data.error || 'No result');
        }
    } catch (e) {
        setCraftResult('Connection error');
    }
    renderCraftPanel();
    updateCraftBtn();
}

// Запрос-ответ через WS с ожиданием ctl-ответа сервера (крафт в игре).
// Резолвится сообщением того же типа ответа (craftLive / craftLiveTake).
let liveCtlResolvers = {};
function requestLive(type, payload) {
    if (!ws || ws.readyState !== WebSocket.OPEN) return Promise.reject(new Error('no ws'));
    return new Promise((resolve, reject) => {
        const list = liveCtlResolvers[type] || (liveCtlResolvers[type] = []);
        list.push(resolve);
        setTimeout(() => {
            const idx = list.indexOf(resolve);
            if (idx >= 0) { list.splice(idx, 1); reject(new Error('timeout')); }
        }, 12000);
        ws.send(JSON.stringify(Object.assign({ type: type }, payload)));
    });
}

// Открыть/закрыть окно крафта.
function toggleCraftPanel() {
    const p = document.getElementById('craftPanel');
    if (!p) return;
    const wasOpen = p.classList.contains('open');
    if (wasOpen) { p.classList.remove('open'); return; }
    p.classList.add('open');
    renderCraftPanel();
}

// Отражение оставшегося результата на кнопке «Крафт».
function updateCraftBtn() {
    const b = document.getElementById('craftBtn');
    if (b) {
        const has = craftPending.length > 0;
        b.classList.toggle('has-pending', has);
        // Меняем только подпись: внутри кнопки ещё и иконка.
        const lbl = document.getElementById('craftBtnLabel');
        if (lbl) lbl.textContent = has ? `Take (${craftPending.length})` : 'Craft';
    }
    renderCraftPanel();
}

document.getElementById('craftBtn').addEventListener('click', toggleCraftPanel);
const craftCloseEl = document.getElementById('craftCloseBtn');
if (craftCloseEl) craftCloseEl.addEventListener('click', () => {
    const p = document.getElementById('craftPanel');
    if (p) p.classList.remove('open');
});

// Главная кнопка окна.
(function initCraftControls() {
    const bi = document.getElementById('craftBtnInner');
    if (bi) bi.addEventListener('click', () => {
        if (craftSpinning) return;
        if (craftPending.length > 0) takeCraft();
        else doCraft();
    });
    const resetBtn = document.getElementById('craftResetBtn');
    if (resetBtn) resetBtn.addEventListener('click', () => {
        if (!craftSpinning) craftResetPile('Wheel reset — shapes returned to inventory.');
    });
})();

// ----- Absorb (уничтожение петалей за XP) в меню -----
// Петали собираются в «кучу» по клику (обычный = +5, Shift = все), затем списываются
// навсегда на сервере, давая опыт. XP растёт в геометрической прогрессии по
// редкости. Превью XP и сравнение уровней клиент считает зеркальной формулой
// (по dmg/maxHp стаков), чтобы не дублировать базовые статы.
const ABSORB_DANGER = [1.0, 1.5, 2.25, 3.375, 5.0625, 7.59375, 11.390625, 17.0859375]; // множители редкости статов (геометрия ×1.5)
const ABSORB_RATIO = 2.0; // геометрический рост XP за редкость: редкость +1 = вдвое больше XP
let absorbPile = []; // [{kind, rarity, count}] — петали, отмеченные на уничтожение

function xpForLevelCli(lvl) { return Math.floor(30 + lvl * 12); }

// XP за ОДИН петаль (зеркало серверного petalStackXP через статы клиентского стака).
// Статы нормализуются на «базовые» (делением на множитель редкости), затем
// база силы × 0.6 × 2^редкость — как на сервере.
function petalXp(it) {
    const rar = Math.min(it.rarity || 0, ABSORB_DANGER.length - 1);
    const mult = ABSORB_DANGER[rar] || 1;
    const baseDmg = (Number(it.dmg) || 0) / mult;
    const baseHp = (Number(it.maxHp) || 0) / mult;
    const power = baseDmg * 2 + baseHp;
    if (power <= 0) return 0;
    return power * 0.6 * Math.pow(ABSORB_RATIO, rar);
}

function absorbPileFor(kind, rarity) {
    let n = 0;
    for (const p of absorbPile) if (p.kind === kind && p.rarity === (rarity || 0)) n += p.count;
    return n;
}

function absorbAvail(it) {
    return Math.max(0, (Number(it.count) || 0) - absorbPileFor(it.kind || '', it.rarity || 0));
}

function absorbPileXP() {
    let t = 0;
    for (const p of absorbPile) {
        const it = (myInventory || []).find(i => (i.kind || '') === p.kind && (i.rarity || 0) === p.rarity);
        if (it) t += petalXp(it) * p.count;
    }
    return Math.floor(t);
}

function setAbsorbResult(msg) {
    const box = document.getElementById('absorbResult');
    if (box) box.textContent = msg;
}

function renderAbsorbPanel() {
    const panel = document.getElementById('absorbPanel');
    if (!panel || !panel.classList.contains('open')) return;
    const pileBox = document.getElementById('absorbPileBox');
    const pileC = document.getElementById('absorbCount');
    const pileI = document.getElementById('absorbInfo');
    const xpP = document.getElementById('absorbXpPreview');
    const cmp = document.getElementById('absorbCompare');
    const bi = document.getElementById('absorbBtnInner');
    const resetBtn = document.getElementById('absorbResetBtn');

    if (pileBox) {
        pileBox.innerHTML = '';
        if (absorbPile.length === 0) {
            const e = document.createElement('div');
            e.className = 'pile-empty';
            e.textContent = 'Drop shapes here to destroy them';
            pileBox.appendChild(e);
        } else {
            for (const p of absorbPile) {
                const it = (myInventory || []).find(i => (i.kind || '') === p.kind && (i.rarity || 0) === p.rarity);
                if (!it) continue;
                const col = rarityColors[Math.min(p.rarity, rarityColors.length - 1)];
                const slot = document.createElement('div');
                slot.className = 'craft-slot';
                slot.style.borderColor = col;
                slot.title = `${it.name || kindName(p.kind)} ${rarityName(p.rarity)} ×${p.count}`;
                const shape = document.createElement('div');
                shape.className = 'petal-shape';
                shape.innerHTML = petalSVG(p.kind, col);
                slot.appendChild(shape);
                const nm = document.createElement('div');
                nm.className = 'inv-name';
                nm.textContent = it.name || kindName(p.kind);
                slot.appendChild(nm);
                const badge = document.createElement('div');
                badge.className = 'cat-count';
                badge.textContent = '×' + p.count;
                slot.appendChild(badge);
                pileBox.appendChild(slot);
            }
        }
    }
    const totalPetal = absorbPile.reduce((s, p) => s + p.count, 0);
    if (pileC) pileC.textContent = 'Added: ' + totalPetal;
    if (pileI) pileI.textContent = absorbPile.length ? (`${absorbPile.length} kind${absorbPile.length > 1 ? 's' : ''}`) : '—';

    const gain = absorbPileXP();
    if (xpP) xpP.textContent = 'XP: +' + gain;
    if (cmp) {
        if (gain <= 0) { cmp.textContent = ''; }
        else {
            let lvl = myLevel, xp = myXP, left = gain;
            while (left > 0) {
                const need = xpForLevelCli(lvl);
                const take = Math.min(left, need - xp);
                xp += take; left -= take;
                if (xp >= need) { xp -= need; lvl++; }
            }
            const lvlText = lvl === myLevel ? `Lv. ${lvl}` : `Lv. ${myLevel} → ${lvl}`;
            cmp.textContent = `${lvlText} · ${Math.floor(myXP)} XP → ${lvl === myLevel ? (Math.floor(myXP) + gain) : Math.floor(xp)} XP`;
        }
    }

    if (resetBtn) resetBtn.disabled = totalPetal === 0;
    if (bi) {
        bi.disabled = totalPetal === 0;
        bi.textContent = totalPetal > 0 ? `Absorb (+${gain} XP)` : 'Absorb';
    }

    const inv = document.getElementById('absorbInventory');
    if (inv) {
        inv.innerHTML = '';
        const totalInv = (myInventory || []).reduce((s, i) => s + (Number(i.count) || 0), 0);
        if (totalInv === 0) {
            const e = document.createElement('div');
            e.className = 'inv-empty';
            e.textContent = 'No shapes in inventory';
            e.style.cssText = 'width:auto;height:40px;font-size:10px;color:#8a93a6;';
            inv.appendChild(e);
        }
        for (const it of (myInventory || [])) {
            const kind = it.kind || '';
            const rarity = it.rarity || 0;
            const avail = absorbAvail(it);
            const col = rarityColors[Math.min(rarity, rarityColors.length - 1)];
            const slot = document.createElement('div');
            slot.className = 'craft-slot' + (avail > 0 ? ' pickable' : '');
            slot.title = avail > 0
                ? `${it.name || kindName(kind)} ${rarityName(rarity)} (available ${avail}) — +${Math.floor(petalXp(it))} XP each\nClick = +1 · Shift = all`
                : 'All shapes in the pile';
            if (avail <= 0) slot.style.opacity = '0.45';
            const shape = document.createElement('div');
            shape.className = 'petal-shape';
            shape.innerHTML = petalSVG(kind, col);
            slot.appendChild(shape);
            const nm = document.createElement('div');
            nm.className = 'inv-name';
            nm.textContent = (it.name || kindName(kind)) + ' ' + rarityName(rarity);
            slot.appendChild(nm);
            const badge = document.createElement('div');
            badge.className = 'cat-count';
            badge.textContent = '×' + avail;
            slot.appendChild(badge);
            if (avail > 0) slot.addEventListener('click', e => absorbAdd(kind, rarity, it.count, e.shiftKey));
            inv.appendChild(slot);
        }
    }
}

function absorbAdd(kind, rarity, total, shift) {
    rarity = rarity || 0;
    const avail = Math.max(0, (Number(total) || 0) - absorbPileFor(kind, rarity));
    if (avail <= 0) { setAbsorbResult('These shapes are already in the pile.'); return; }
    const add = shift ? avail : Math.min(1, avail);
    const existing = absorbPile.find(p => p.kind === kind && p.rarity === rarity);
    if (existing) existing.count += add;
    else absorbPile.push({ kind, rarity, count: add });
    setAbsorbResult('');
    renderAbsorbPanel();
}

function absorbResetPile() {
    absorbPile = [];
    setAbsorbResult('Pile reset — shapes returned to inventory.');
    renderAbsorbPanel();
}

async function doAbsorb() {
    if (absorbPile.length === 0) return;
    const items = absorbPile.map(p => ({ kind: p.kind, rarity: p.rarity, count: p.count }));
    const btn = document.getElementById('absorbBtnInner');
    const prev = btn ? btn.textContent : '';
    if (btn) { btn.disabled = true; btn.textContent = '…'; }
    try {
        const data = await apiCall('/api/absorb', { token: authToken, items });
        if (data.ok) {
            applyProfile(data); // обновляет уровень/XP (и инвентарь, если прислан)
            // Если сервер опустил `inventory` при опустошении (omitempty) — его
            // `applyProfile` не заменил; убираем поглощённые петали локально,
            // иначе они останутся в меню Absorb.
            if (!Array.isArray(data.inventory)) {
                for (const p of items) {
                    const idx = (myInventory || []).findIndex(i => (i.kind || '') === p.kind && (i.rarity || 0) === p.rarity);
                    if (idx >= 0) {
                        const it = myInventory[idx];
                        if ((it.count || 0) > p.count) it.count -= p.count;
                        else myInventory.splice(idx, 1);
                    }
                }
                lastInvSig = '';
            }
            absorbPile = [];
            const gain = data.xpAdd || 0;
            setAbsorbResult(gain > 0 ? `+${gain} XP` : 'Done');
            addSystemToast(gain > 0 ? `Absorb: +${gain} XP` : 'Absorb done', '');
            // Сверяемся с сервером, чтобы и обычное меню показывало актуальное.
            loadProfile();
            renderAbsorbPanel();
        } else {
            setAbsorbResult(data.error || 'Absorb error');
            if (btn) btn.textContent = prev;
        }
    } catch (e) {
        setAbsorbResult('Connection error');
        if (btn) btn.textContent = prev;
    }
    renderAbsorbPanel();
}

function toggleAbsorbPanel() {
    const p = document.getElementById('absorbPanel');
    if (!p) return;
    const wasOpen = p.classList.contains('open');
    if (wasOpen) { p.classList.remove('open'); return; }
    p.classList.add('open');
    renderAbsorbPanel();
}

document.getElementById('absorbBtn').addEventListener('click', toggleAbsorbPanel);
const absorbCloseEl = document.getElementById('absorbCloseBtn');
if (absorbCloseEl) absorbCloseEl.addEventListener('click', () => {
    const p = document.getElementById('absorbPanel');
    if (p) p.classList.remove('open');
});
(function initAbsorbControls() {
    const bi = document.getElementById('absorbBtnInner');
    if (bi) bi.addEventListener('click', doAbsorb);
    const resetBtn = document.getElementById('absorbResetBtn');
    if (resetBtn) resetBtn.addEventListener('click', absorbResetPile);
})();

loginBtn.addEventListener('click', async () => {
    const nick = loginNick.value.trim();
    const pass = loginPass.value;
    if (!nick || !pass) { authError.textContent = 'Fill in all fields'; return; }
    authError.textContent = '';
    try {
        const data = await apiCall('/api/login', { nick, password: pass });
        console.log('LOGIN response:', data);
        if (!data.ok) { authError.textContent = data.error; return; }
        authToken = data.token;
        localStorage.setItem('authToken', authToken);
        myNickname = nick;
        myInventory = [];
        console.log('LOGIN ok, showing menu');
        showScreen('menu');
        startOnlinePolling();
        loadProfile();
    } catch (e) {
        console.error('LOGIN fetch error:', e);
        authError.textContent = 'Connection to server failed';
    }
});

regBtn.addEventListener('click', async () => {
    const nick = regNick.value.trim();
    const pass = regPass.value;
    if (!nick || !pass) { regError.textContent = 'Fill in all fields'; return; }
    regError.textContent = '';
    try {
        const data = await apiCall('/api/register', { nick, password: pass });
        console.log('REGISTER response:', data);
        if (!data.ok) { regError.textContent = data.error; return; }
        authToken = data.token;
        localStorage.setItem('authToken', authToken);
        myNickname = nick;
        myInventory = [];
        console.log('REGISTER ok, showing menu');
        showScreen('menu');
        startOnlinePolling();
        loadProfile();
    } catch (e) {
        console.error('REGISTER fetch error:', e);
        regError.textContent = 'Connection to server failed';
    }
});

loginNick.addEventListener('keydown', (e) => { if (e.key === 'Enter') loginBtn.click(); });
loginPass.addEventListener('keydown', (e) => { if (e.key === 'Enter') loginBtn.click(); });
regNick.addEventListener('keydown', (e) => { if (e.key === 'Enter') regBtn.click(); });
regPass.addEventListener('keydown', (e) => { if (e.key === 'Enter') regBtn.click(); });

document.getElementById('toRegister').addEventListener('click', () => { authError.textContent = ''; showScreen('register'); });
document.getElementById('toLogin').addEventListener('click', () => { regError.textContent = ''; showScreen('login'); });

// ----- Онлайн-счётчик: раз в 15 сек отмечаемся на /api/online и читаем totals.
// Сервер считает сидящих в меню по heartbeat (окно 45 сек), в игре — по WS.
let onlineTimer = null;
async function refreshOnline() {
    if (!authToken) return;
    try {
        const data = await apiCall('/api/online', { token: authToken });
        if (data && data.ok) {
            const box = document.getElementById('onlineBox');
            // Панель показываем только в меню (в игре её прячет syncTicker).
            if (box && menuOpen()) {
                box.style.display = 'block';
                const t = document.getElementById('onlineTotal');
                const d = document.getElementById('onlineDetail');
                if (t) t.textContent = 'Online: ' + data.total;
                if (d) d.textContent = 'in game ' + data.inGame + ' · in menu ' + data.inMenu;
            }
        }
    } catch (e) { /* сервер недоступен — просто пропускаем опрос */ }
}
function startOnlinePolling() {
    if (!authToken) return;
    refreshOnline();
    if (onlineTimer) clearInterval(onlineTimer);
    onlineTimer = setInterval(refreshOnline, 15000);
}
function stopOnlinePolling() {
    if (onlineTimer) { clearInterval(onlineTimer); onlineTimer = null; }
    const box = document.getElementById('onlineBox');
    if (box) box.style.display = 'none';
}

document.getElementById('logoutBtn').addEventListener('click', () => {
    authToken = '';
    localStorage.removeItem('authToken');
    stopOnlinePolling();
    intentionalClose = true;
    if (connectTimer) { clearTimeout(connectTimer); connectTimer = null; }
    if (ws) { try { ws.close(); } catch (e) {} }
    ws = null;
    showScreen('login');
});

// Показать нужный экран при загрузке — вызывается в конце файла,
// после объявления всех переменных (ws и пр.).
// ----- PixiJS -----
const app = new PIXI.Application({
    width: window.innerWidth,
    height: window.innerHeight,
    backgroundColor: 0x1a1e27,
    resolution: window.devicePixelRatio || 1,
    autoDensity: true,
});
document.getElementById('game').appendChild(app.view);

const world = new PIXI.Container();
app.stage.addChild(world);

// Линейка/сетка (админ): накладывается поверх фона зон, но под игровыми объектами
// (порядок выставляется в bringObjectsAbove).
const gridLayer = new PIXI.Container();
gridLayer.visible = false;
world.addChild(gridLayer);
let gridOn = false;
const gridStep = 100; // шаг сетки в пикселях мира

let zones = [];
let portals = [];
const actors = {};
const mobActors = {};
const petalActors = {};
const groundActors = {};
const projectileActors = {};
// Живые saturn-петали по игрокам: id -> редкость (заполняется в renderPetals
// из состояний петалей; кольцо-аура рисуем у ВСЕХ игроков с живым сатурном).
let saturnAlive = {};
// Кулдаун петалей в хотбаре (по слоту активной линейки): доля оставшейся
// перезарядки 0..1, обновляется в renderPetals из id "pet_<ник>_h<слот>".
const hotbarSlotCD = {};
const mobFades = new Map();       // id -> {c, t} тающими «трупами» мобов
const groundFades = new Map();    // id -> {c, t} «взлетающими и растворяющимися» дропами
const zoneSprites = [];
const portalContainers = [];

let myId = null;
let myZone = -1;
let lastMaxHP = 50;
let myPlayerX = 0, myPlayerY = 0; // позиция моего игрока (для анимации щита)

// ----- WebSocket -----
let ws = null;
let gameStarted = false;
let waveModeClient = false; // волновой режим (круглая арена): экраны смерти/лобби
let connectTimer = null;   // таймер авто-переподключения
let intentionalClose = false; // true, когда закрытие соединения было намеренным (меню/выход)

// Отправка сообщения на сервер (если соединение открыто).
function sendMsg(type, payload) {
    if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify(Object.assign({ type }, payload || {})));
    }
}

function connect() {
    // Не открываем второе соединение, если уже подключены или в процессе.
    if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) return;
    if (!ws || ws.readyState === WebSocket.CLOSED) {
        // ws закрыт по ошибке — это авто-переподключение после обрыва.
        intentionalClose = false;
    }
    const proto = location.protocol === 'https:' ? 'wss' : 'ws';
    ws = new WebSocket(`${proto}://${location.host}/ws`);

    ws.onopen = () => {
        ws.send(JSON.stringify({ type: 'hello', token: authToken, color: chosenColor, code: getLobbyCode() }));
    };

    ws.onmessage = (event) => {
        const msg = JSON.parse(event.data);
        switch (msg.type) {
            case 'joined':
                myId = msg.id;
                gameStarted = true;
                updatePlayerAvatar();
                setUpAdmin(!!msg.admin);
                setAdminState(!!msg.god, !!msg.invisible, !!msg.passive, (msg.team || 0), !!msg.oneshot, msg.size);
                hideLoading();
                const exBtn = document.getElementById('exitToMenuBtn');
                if (exBtn) exBtn.style.display = 'flex';
                break;
            case 'adminState':
                setAdminState(!!msg.god, !!msg.invisible, !!msg.passive, (msg.team || 0), !!msg.oneshot, msg.size);
                break;
            case 'world':
                buildWorld(msg);
                break;
            case 'transfer':
                startTransfer(msg.from, msg.to);
                break;
            case 'state':
                render(msg);
                break;
            case 'died':
                showDeath();
                break;
            case 'mobDied':
                spawnMobFade(msg);
                break;
            case 'chat':
                addChatMessage(msg.name || '?', msg.text || '', msg.zone || '', msg.id || '');
                break;
            case 'pickedUp':
                pushToast('pickup', `+ ${kindName(msg.kind)} ${msg.rarity || ''}`);
                break;
            case 'sys':
                addSystemToast(msg.text || '', msg.kind || '');
                break;
            case 'compass':
                // {on, a (мировой угол), d} — направление компаса на ближайшего
                // Fractal/Chaos-моба (работает только на 2-м уровне, зоны 8-15).
                compassInfo = msg;
                break;
            case 'wave':
                handleWaveMsg(msg);
                break;
            case 'adminPlayers':
                renderAdminPlayers(msg.players || []);
                break;
            case 'adminAck':
                if (adminAckEl) adminAckEl.textContent = msg.text || '';
                break;
            case 'adminMobs':
                renderAdminMobs(msg.zones || []);
                break;
            case 'levelup':
                myLevel = msg.level;
                updateMenuLevel();
                updateXP(myLevel, myXP, myXPMax);
                break;
            case 'craftLive':
            case 'craftLiveTake': {
                // Ответ на внутриигровой крафт — резолвим requestLive.
                const list = liveCtlResolvers[msg.type];
                if (list && list.length > 0) {
                    const fn = list.shift();
                    try { fn(msg); } catch (e) {}
                }
                break;
            }
            case 'error':
                sellBackToMenu(msg.message);
                break;
        }
    };

    ws.onclose = () => {
        if (connectTimer) { clearTimeout(connectTimer); connectTimer = null; }
        ws = null;
        // Выход в меню инициирован — финишируем сразу.
        if (pendingMenuExit) { enterMenuFromWorld(); return; }
        // Переподключаемся пока есть открытая сессия (меню или игра), если закрытие не намеренное.
        if (authToken && !intentionalClose) {
            connectTimer = setTimeout(connect, 2000);
        }
    };
}

function sellBackToMenu(message) {
    gameStarted = false;
    intentionalClose = true;
    myId = null;
    if (connectTimer) { clearTimeout(connectTimer); connectTimer = null; }
    if (ws) { try { ws.close(); } catch (e) {} }
    if (message && message.includes('token')) {
        authToken = '';
        localStorage.removeItem('authToken');
        showScreen('login');
    } else {
        showScreen('menu');
    }
    errorBox.textContent = message || '';
}

// ----- Построение мира -----
// ----- Линейка/сетка -----
// Перерисовывает сетку по всем зонам. Вызывается при включении и при buildWorld.
function rebuildGrid() {
    gridLayer.removeChildren();
    if (!gridOn) { gridLayer.visible = false; return; }
    gridLayer.visible = true;
    for (const z of zones) {
        const g = new PIXI.Graphics();
        g.lineStyle(1, 0xffffff, 0.10);
        // Вертикальные линии.
        for (let x = gridStep; x < z.w; x += gridStep) {
            g.moveTo(x, 0); g.lineTo(x, z.h);
        }
        // Горизонтальные линии.
        for (let y = gridStep; y < z.h; y += gridStep) {
            g.moveTo(0, y); g.lineTo(z.w, y);
        }
        // Границы зоны — ярче.
        g.lineStyle(2, 0xffffff, 0.35);
        g.drawRect(0, 0, z.w, z.h);
        g.position.set(z.x, z.y);
        gridLayer.addChild(g);
    }
}

function toggleGrid() {
    gridOn = !gridOn;
    rebuildGrid();
    return gridOn;
}

// ----- Тематические узоры биомов (векторные, кодом PIXI) -----
// Детерминированный генератор случайности по сиду (чтобы узор не прыгал при rebuild).
function makeRng(seed) {
    let s = seed;
    return () => {
        s = (s * 1103515245 + 12345) & 0x7fffffff;
        return s / 0x7fffffff;
    };
}

function buildPolyPath(g, cx, cy, r, n, rot) {
    g.moveTo(cx + r * Math.cos(rot), cy + r * Math.sin(rot));
    for (let i = 1; i <= n; i++) {
        const a = rot + i * (2 * Math.PI / n);
        g.lineTo(cx + r * Math.cos(a), cy + r * Math.sin(a));
    }
}

// Звезда: внешний радиус R, внутренний r, pointss вершин, поворот rot.
function buildStar(g, cx, cy, R, r, points, rot) {
    const step = Math.PI / points;
    for (let i = 0; i < points * 2; i++) {
        const rad = (i % 2 === 0) ? R : r;
        const a = rot + i * step;
        const x = cx + rad * Math.cos(a);
        const y = cy + rad * Math.sin(a);
        if (i === 0) g.moveTo(x, y); else g.lineTo(x, y);
    }
    g.closePath();
}

// Ромб: габариты w,h по осям, поворот rot.
function drawRhombus(g, cx, cy, w, h, rot) {
    const pts = [[w / 2, 0], [0, h / 2], [-w / 2, 0], [0, -h / 2]];
    for (let i = 0; i < pts.length; i++) {
        const [px, py] = pts[i];
        const x = cx + px * Math.cos(rot) - py * Math.sin(rot);
        const y = cy + px * Math.sin(rot) + py * Math.cos(rot);
        if (i === 0) g.moveTo(x, y); else g.lineTo(x, y);
    }
    g.closePath();
}

// Рисует геометрический узор биома на полу зоны. Вызывается при buildWorld.
// drawSafeZoneFigures заполняет безопасную полосу зоны светлыми зелёными
// фигурами: рассеянными точками, треугольниками, ромбами и звёздочками.
function drawSafeZoneFigures(g, z, safeTop) {
    const rng = makeRng(z.index * 1337 + 731);
    const greens = [0x98f3bd, 0x7dffb0, 0xb8ffd6, 0x5eea9c, 0xd0ffe6];
    const col = () => greens[Math.floor(rng() * greens.length)];
    const n = Math.round(z.w / 130);
    const bandH = z.h - safeTop;
    for (let i = 0; i < n; i++) {
        const cx = rng() * z.w;
        const cy = safeTop + 18 + rng() * (bandH - 36);
        const s = 6 + rng() * 10;
        const kind = Math.floor(rng() * 4);
        const a = 0.35 + rng() * 0.4;
        if (kind === 0) {
            g.lineStyle(0);
            g.beginFill(col(), a);
            g.drawCircle(cx, cy, 2 + rng() * 4);
            g.endFill();
        } else if (kind === 1) {
            g.lineStyle(1.5, col(), a * 0.9);
            buildPolyPath(g, cx, cy, s, 3, rng() * Math.PI);
        } else if (kind === 2) {
            g.lineStyle(1.5, col(), a * 0.9);
            drawRhombus(g, cx, cy, s, s * (0.6 + rng() * 0.5), rng() * Math.PI);
        } else {
            g.lineStyle(1.5, col(), a * 0.9);
            buildStar(g, cx, cy, s, s * 0.45, 5, rng() * Math.PI);
        }
    }
    g.lineStyle(0);
}

function drawBiomeDecoration(g, z) {
    const rng = makeRng(z.index * 7919 + 13);
    const light = 0xffffff;
    switch (z.biome) {
        case 0: // Point — россыпь светящихся точек
            g.lineStyle(0);
            for (let x = 120; x < z.w; x += 190) {
                for (let y = 120; y < z.h; y += 190) {
                    const r = 3 + rng() * 5;
                    const a = 0.18 + rng() * 0.25;
                    g.beginFill(light, a);
                    g.drawCircle(x + (rng() - 0.5) * 80, y + (rng() - 0.5) * 80, r);
                    g.endFill();
                }
            }
            break;
        case 1: // Line — вертикальные лучи + редкие диагонали
            g.lineStyle(2, light, 0.10);
            for (let x = 0; x < z.w; x += 140) {
                const wob = Math.sin(z.index * 3 + x * 0.001) * 40;
                g.moveTo(x, 0);
                g.lineTo(x + wob, z.h);
            }
            g.lineStyle(3, light, 0.18);
            for (let i = 0; i < 18; i++) {
                const x0 = rng() * z.w, y0 = rng() * z.h;
                const len = 300 + rng() * 500;
                const ang = rng() * Math.PI * 2;
                g.moveTo(x0, y0);
                g.lineTo(x0 + Math.cos(ang) * len, y0 + Math.sin(ang) * len);
            }
            break;
        case 2: // Triangle — россыпь треугольников
            g.lineStyle(2, light, 0.12);
            for (let i = 0; i < 80; i++) {
                const cx = rng() * z.w, cy = rng() * z.h;
                const r = 30 + rng() * 100;
                g.moveTo(cx + r, cy);
                buildPolyPath(g, cx, cy, r, 3, rng() * Math.PI);
            }
            break;
        case 3: // Square — сетка квадратов
            g.lineStyle(2, light, 0.10);
            for (let x = 80; x < z.w; x += 200) {
                for (let y = 80; y < z.h; y += 200) {
                    const s = 30 + rng() * 60;
                    const rot = rng() * 0.4;
                    g.drawRect(x - s / 2 + (rng() - 0.5) * 40, y - s / 2 + (rng() - 0.5) * 40, s, s);
                }
            }
            break;
        case 4: // Pentagon — разброс пятиугольников
            g.lineStyle(2, light, 0.12);
            for (let i = 0; i < 60; i++) {
                const cx = rng() * z.w, cy = rng() * z.h;
                const r = 35 + rng() * 90;
                g.moveTo(cx + r, cy);
                buildPolyPath(g, cx, cy, r, 5, rng() * Math.PI);
            }
            break;
        case 5: // Hexagon — соты (гексагональная решётка)
            g.lineStyle(1.5, light, 0.08);
            const hr = 60;
            const h = hr * Math.sqrt(3);
            for (let row = -1; row < z.h / h + 1; row++) {
                for (let col = -1; col < z.w / (hr * 1.5) + 1; col++) {
                    const cx = col * hr * 3;
                    const cy = row * h + (col % 2 === 0 ? 0 : h / 2);
                    g.moveTo(cx + hr, cy);
                    buildPolyPath(g, cx, cy, hr, 6, 0);
                }
            }
            break;
        case 6: // Fractal — вложенные самоподобные композиции
            for (let i = 0; i < 95; i++) {
                const cx = rng() * z.w, cy = rng() * z.h;
                const r0 = 55 + rng() * 180;
                const levels = 5 + Math.floor(rng() * 4);
                const rot = rng() * Math.PI;
                for (let L = 0; L < levels; L++) {
                    const r = r0 * (1 - L / levels);
                    g.lineStyle(2, light, 0.05 + L * 0.035);
                    buildPolyPath(g, cx, cy, r, 3 + (L % 3), rot + L * 0.25);
                }
            }
            for (let i = 0; i < 300; i++) {
                const cx = rng() * z.w, cy = rng() * z.h;
                const r = 5 + rng() * 16;
                g.lineStyle(1, light, 0.06 + rng() * 0.05);
                buildPolyPath(g, cx, cy, r, 3, rng() * Math.PI);
            }
            break;
        case 7: // Chaos — хаотичный разброс разноцветных фигур
            const palette = [0xff4d6d, 0xffd54f, 0x4aa3ff, 0xb48cff, 0x3ddc84, 0xff9f43, 0x00e5ff, 0xff5c8a, 0xa6ff6b];
            const col = () => palette[Math.floor(rng() * palette.length)];
            for (let i = 0; i < 260; i++) {
                const cx = rng() * z.w, cy = rng() * z.h;
                const rot = rng() * Math.PI * 2;
                const size = 14 + rng() * 72;
                const c = col();
                const kind = Math.floor(rng() * 6);
                if (kind === 0) {
                    const r = 3 + rng() * 15;
                    g.lineStyle(0);
                    g.beginFill(c, 0.55 + rng() * 0.45);
                    g.drawCircle(cx, cy, r);
                    g.endFill();
                } else if (kind === 1) {
                    g.lineStyle(2.5, c, 0.8 + rng() * 0.2);
                    buildStar(g, cx, cy, size, size * (0.4 + rng() * 0.3), 5, rot);
                } else if (kind === 2) {
                    g.lineStyle(2.5, c, 0.8 + rng() * 0.2);
                    drawRhombus(g, cx, cy, size, size * (0.6 + rng() * 0.9), rot);
                } else if (kind === 3) {
                    const len = 40 + rng() * 170;
                    const a = rng() * Math.PI * 2;
                    g.lineStyle(3, c, 0.8 + rng() * 0.2);
                    g.moveTo(cx - Math.cos(a) * len / 2, cy - Math.sin(a) * len / 2);
                    g.lineTo(cx + Math.cos(a) * len / 2, cy + Math.sin(a) * len / 2);
                } else if (kind === 4) {
                    g.lineStyle(2.5, c, 0.8 + rng() * 0.2);
                    buildPolyPath(g, cx, cy, size, 3 + Math.floor(rng() * 6), rot);
                } else {
                    g.lineStyle(3, c, 0.75);
                    const n2 = size * 0.6;
                    g.moveTo(cx - n2, cy - n2); g.lineTo(cx + n2, cy + n2);
                    g.moveTo(cx + n2, cy - n2 * 0.6); g.lineTo(cx - n2 * 0.6, cy + n2);
                }
            }
            g.lineStyle(0);
            for (let i = 0; i < 520; i++) {
                const r = 1.5 + rng() * 4;
                g.beginFill(col(), 0.4 + rng() * 0.6);
                g.drawCircle(rng() * z.w, rng() * z.h, r);
                g.endFill();
            }
            break;
    }
}

function buildWorld(msg) {
    zones = msg.zones;
    portals = msg.portals;

    for (const s of zoneSprites) world.removeChild(s);
    for (const c of portalContainers) world.removeChild(c);
    for (const [id, f] of mobFades) { world.removeChild(f.c); }
    mobFades.clear();
    for (const [id, f] of groundFades) { world.removeChild(f.c); }
    groundFades.clear();
    zoneSprites.length = 0;
    portalContainers.length = 0;

    // Волновой режим: круглый «басейн» с ПОЛНЫМ оформлением выбранной зоны —
    // цвет и тематический узор биома те же, что у этой зоны в основном мире,
    // обрезанные кругом арены (маска). Редкости мобов растут, размер — нет.
    if (msg.wave) {
        waveModeClient = true;
        portals = [];
        const az = zones[msg.arenaZone] || zones[0] || { x: 0, y: 0, w: 8000, h: 5000, color: 0x2ecc71, biome: 0, index: 0 };
        const c = new PIXI.Container();
        const deco = new PIXI.Graphics();
        deco.position.set(az.x, az.y);
        const baseCol = az.biome === 7 ? 0x000000 : az.color;
        const baseA = az.biome === 7 ? 1 : 0.30;
        deco.beginFill(baseCol, baseA);
        deco.drawRect(0, 0, az.w, az.h);
        deco.endFill();
        // Тот же узор биома, что рисуется у зоны в основном мире.
        drawBiomeDecoration(deco, az);
        c.addChild(deco);
        // Круглая маска арены (в локальных координатах контейнера).
        const mask = new PIXI.Graphics();
        mask.beginFill(0xffffff);
        mask.drawCircle(msg.arenaX, msg.arenaY, msg.arenaR);
        c.addChild(mask);
        c.mask = mask;
        world.addChild(c);
        zoneSprites.push(c);
        // Кромка арены — тёмный корпус + светящаяся внутренняя линия.
        const ring = new PIXI.Graphics();
        const edgeCol = shade ? shade(az.biome === 7 ? 0x333333 : az.color, -70) : 0x2f7a3e;
        ring.lineStyle(12, edgeCol, 1);
        ring.drawCircle(msg.arenaX, msg.arenaY, msg.arenaR);
        ring.lineStyle(3, az.color, 0.9);
        ring.drawCircle(msg.arenaX, msg.arenaY, msg.arenaR - 8);
        world.addChild(ring);
        zoneSprites.push(ring);
        return;
    }
    waveModeClient = false;
    // Зоны: цветные прямоугольники + рамка-стена + тематический узор биома.
    for (let zi = 0; zi < zones.length; zi++) {
        const z = zones[zi];
        const g = new PIXI.Graphics();
        // Chaos (биом 7) — чисто чёрный фон, как настоящий хаос.
        const baseCol = z.biome === 7 ? 0x000000 : z.color;
        const baseA = z.biome === 7 ? 1 : 0.30;
        const wallCol = z.biome === 7 ? 0x1a1a1a : z.color;
        // Зоны II со скруглёнными углами (cornerR>0): пол и стены рисуем одним
        // скруглённым прямоугольником — стены читаются как 8px линия по периметру,
        // а карманы прямых углов исчезают (в них застревали крупные мобы).
        if (z.cornerR > 0) {
            g.beginFill(baseCol, baseA);
            g.lineStyle(8, wallCol, 0.90);
            g.drawRoundedRect(0, 0, z.w, z.h, z.cornerR);
            g.lineStyle(0);
            g.endFill();
        } else {
            g.beginFill(baseCol, baseA);
            g.drawRect(0, 0, z.w, z.h);
            g.endFill();
            g.beginFill(wallCol, 0.90);
            g.drawRect(-4, 0, 8, z.h); // левая стена
            g.drawRect(z.w - 4, 0, 8, z.h); // правая стена
            g.drawRect(0, -4, z.w, 8); // верхняя стена
            g.drawRect(0, z.h - 4, z.w, 8); // нижняя стена
            g.endFill();
        }

        // Safe Zone внизу зоны: светло-зелёная тематическая полоса с фигурами.
        // Полоса делается ПОД узором биома, а фигуры — ПОВЕРХ (чтобы узор зоны
        // остался видимым, но Safe Zone читалась как «безопасная»).
        if (z.safeH > 0) {
            const safeTop = z.h - z.safeH;
            g.beginFill(0xa8f5c8, 0.16);
            g.drawRect(0, safeTop, z.w, z.safeH);
            g.endFill();
            // Верхняя граница Safe Zone — мягкая светящаяся линия.
            g.lineStyle(2, 0x7dffb0, 0.35);
            g.moveTo(0, safeTop);
            g.lineTo(z.w, safeTop);
            g.lineStyle(0);

            // Чекпоинт-площадка: ВИЗУАЛ УБРАН (по просьбе игрока), функционал
            // закрепления чекпоинта на сервере остаётся прежним.
            if (false && zi < 8) {
                const padW = 800;
                const px0 = z.w / 2 - padW / 2;
                g.beginFill(0x2ecc71, 0.10);
                g.drawRect(px0, safeTop, padW, z.safeH);
                g.endFill();
                g.lineStyle(2, 0x2ecc71, 0.5);
                g.drawRoundedRect(px0 + 6, safeTop + 6, padW - 12, z.safeH - 12, 14);
                g.lineStyle(0);

                // Карман безопасности у портала вниз — только в L2-зонах (низ).
                if (zi >= 8) {
                    const pkH = 380;
                    const pkY = z.h - pkH;
                    g.beginFill(0x4a9bff, 0.10);
                    g.drawRect(z.w / 2 - 300, pkY, 600, pkH);
                    g.endFill();
                    g.lineStyle(2, 0x4a9bff, 0.45);
                    g.drawRoundedRect(z.w / 2 - 300 + 6, pkY + 6, 600 - 12, pkH - 12, 14);
                    g.lineStyle(0);
                }
            }
        }

        // Тематический узор биома поверх фона (дизайн зоны не ломаем).
        drawBiomeDecoration(g, z);

        // Фигуры Safe Zone — светлая зелёная тематика: точки, сердечки,
        // звёздочки, ромбы, рассеянные по безопасной полосе.
        if (z.safeH > 0) {
            const safeTop = z.h - z.safeH;
            drawSafeZoneFigures(g, z, safeTop);
        }

        // Разделительная стена между Safe Zone и Mob Zone — ВИДИМАЯ,
        // массивная и красивая: тёмный корпус с кирпичной кладкой, светящиеся
        // зелёные кромки и колонны по краям прохода. Стена рисуется поверх
        // узора биома, чтобы её было хорошо видно.
        if (z.safeH > 0 && z.wallTh > 0 && z.wallGap > 0) {
            const safeTop = z.h - z.safeH;
            const wy = safeTop - z.wallTh;
            const cx = z.w / 2;
            const half = z.wallGap / 2;
            const wallFill = z.biome === 7 ? 0x171b22 : 0x252d3a;
            const brickH = z.wallTh / 4;
            const segs = [[0, cx - half], [cx + half, z.w]];
            for (const [sx0, sx1] of segs) {
                // Корпус стены.
                g.beginFill(wallFill, 1);
                g.drawRect(sx0, wy, sx1 - sx0, z.wallTh);
                g.endFill();
                // Объём: светлая верхушка + тень у основания.
                g.beginFill(0xffffff, 0.07);
                g.drawRect(sx0, wy, sx1 - sx0, brickH);
                g.endFill();
                g.beginFill(0x000000, 0.30);
                g.drawRect(sx0, wy + z.wallTh * 0.82, sx1 - sx0, z.wallTh * 0.18);
                g.endFill();
                // Кирпичная кладка: ряды + вертикальные швы со сдвигом.
                g.lineStyle(1, 0x0b0e12, 0.9);
                for (let row = 1; row < 4; row++) {
                    const y = wy + row * brickH;
                    g.moveTo(sx0, y);
                    g.lineTo(sx1, y);
                }
                for (let row = 0; row < 4; row++) {
                    const yLo = wy + row * brickH;
                    const yHi = wy + (row + 1) * brickH;
                    for (let bx = sx0; bx < sx1; bx += 90) {
                        const off = (row % 2) ? 45 : 0;
                        if (bx + off < sx1) {
                            g.moveTo(bx + off, yLo);
                            g.lineTo(bx + off, yHi);
                        }
                    }
                }
                g.lineStyle(0);
                // Верхняя светящаяся зелёная кромка стены.
                g.lineStyle(3, 0x7dffb0, 0.85);
                g.moveTo(sx0, wy + 1);
                g.lineTo(sx1, wy + 1);
                g.lineStyle(0);
                // Нижняя кромка — граница Safe Zone.
                g.lineStyle(2, 0x5eea9c, 0.55);
                g.moveTo(sx0, safeTop);
                g.lineTo(sx1, safeTop);
                g.lineStyle(0);
            }
            // Колонны по краям прохода.
            for (const px of [cx - half, cx + half]) {
                g.beginFill(0x7fdda8, 0.9);
                g.drawRect(px - 10, wy - 8, 20, z.wallTh + 8);
                g.endFill();
                g.beginFill(0x0b0e12, 0.45);
                g.drawRect(px - 10, wy - 8, 6, z.wallTh + 8);
                g.endFill();
            }
            // Тропа-порог в проходе: мягкая светящаяся подложка.
            g.beginFill(0x7dffb0, 0.16);
            g.drawRect(cx - half, safeTop - brickH, half * 2, brickH);
            g.endFill();
            g.lineStyle(2, 0x7dffb0, 0.35);
            g.moveTo(cx - half, safeTop - brickH);
            g.lineTo(cx + half, safeTop - brickH);
            g.lineStyle(0);
        }

        // Конфигурируемые стены зоны (редактор walls). Рисуются поверх всего,
        // чтобы были видны сразу: тёмный кирпич + оранжевая кромка сверху.
        if (z.walls && z.walls.length) {
            for (const wl of z.walls) {
                const brickRows = 4;
                const brickH = Math.max(8, wl.h / brickRows);
                const fill = z.biome === 7 ? 0x241318 : 0x3a2b30;
                g.beginFill(fill, 1);
                g.drawRect(wl.x, wl.y, wl.w, wl.h);
                g.endFill();
                g.beginFill(0xffffff, 0.07);
                g.drawRect(wl.x, wl.y, wl.w, brickH);
                g.endFill();
                g.beginFill(0x000000, 0.30);
                g.drawRect(wl.x, wl.y + wl.h * 0.82, wl.w, wl.h * 0.18);
                g.endFill();
                g.lineStyle(1, 0x12080b, 0.8);
                for (let row = 1; row < brickRows; row++) {
                    const y = wl.y + row * brickH;
                    g.moveTo(wl.x, y);
                    g.lineTo(wl.x + wl.w, y);
                }
                for (let row = 0; row < brickRows; row++) {
                    const yLo = wl.y + row * brickH;
                    const yHi = wl.y + (row + 1) * brickH;
                    const off = (row % 2) ? Math.min(45, wl.w / 2) : 0;
                    for (let bx = wl.x; bx < wl.x + wl.w; bx += 90) {
                        if (bx + off < wl.x + wl.w) {
                            g.moveTo(bx + off, yLo);
                            g.lineTo(bx + off, yHi);
                        }
                    }
                }
                g.lineStyle(0);
                g.lineStyle(2, 0xff9c6a, 0.55);
                g.moveTo(wl.x, wl.y);
                g.lineTo(wl.x + wl.w, wl.y);
                g.lineStyle(0);
            }
        }

        g.position.set(z.x, z.y);
        world.addChild(g);
        zoneSprites.push(g);
    }

    // Порталы - лежащие светящиеся кольца с мерцающими частицами.
    for (const p of portals) {
        portalContainers.push(makePortal(p));
    }

    // Фон/декорации (зоны и порталы) перерисованы первыми; возвращаем игровые
    // объекты (игроки, мобы, петали, предметы) наверх, чтобы ничего не открывало их.
    bringObjectsAbove();
    if (gridOn) rebuildGrid();
}

// Переупорядочивает детей world так, чтобы все объекты игры рисовались выше
// декоративного слоя (зоны/порталы), независимо от порядка их добавления.
function bringObjectsAbove() {
    const zg = zoneSprites.slice();
    const pg = portalContainers.slice();
    const obj = [];
    for (const k of Object.keys(actors)) obj.push(actors[k].sprite);
    for (const k of Object.keys(mobActors)) obj.push(mobActors[k].c);
    for (const k of Object.keys(petalActors)) obj.push(petalActors[k].g);
    for (const k of Object.keys(groundActors)) obj.push(groundActors[k].c);
    for (const el of obj) world.removeChild(el);

    // Убираем из дерева все фоновые графики, затем кладём их наверх фонами.
    for (let i = 0; i < world.children.length; ) {
        const ch = world.children[i];
        if (zg.includes(ch) || pg.includes(ch)) {
            world.removeChild(ch);
            continue;
        }
        i++;
    }
    for (const el of zg) world.addChild(el);
    for (const el of pg) world.addChild(el);
    world.addChild(gridLayer); // сетка — под игровыми объектами, над фоном
    for (const el of obj) world.addChild(el);
}

// makePortal создаёт красивый лежащий кольцевой портал: светящаяся воронка,
// вращающаяся спираль, переливы и мерцающие частицы.
function makePortal(p) {
    const dest = zones[p.to];
    const col = dest ? dest.color : 0xffffff;
    const R = p.w / 2;

    const holder = new PIXI.Container();
    const t = { p, R, col, phase: Math.random() * 100 };
    holder.userData = t;

    // 1) Мягкое внешнее свечение: несколько концентрических колец с убывающей прозрачностью.
    const glow = new PIXI.Graphics();
    for (let i = 6; i >= 1; i--) {
        glow.beginFill(col, 0.03 + i * 0.03);
        glow.drawCircle(0, 0, R + i * 8);
        glow.endFill();
    }
    holder.addChild(glow);
    t.glow = glow;

    // 2) Яркий обод портала (граница кольца).
    const ring = new PIXI.Graphics();
    ring.lineStyle(R * 0.10, 0xffffff, 1);
    ring.drawCircle(0, 0, R);
    holder.addChild(ring);
    t.ring = ring;

    // 3) Внутренняя "поверхность" воронки: слои переливающегося цвета.
    const surface = new PIXI.Graphics();
    surface.beginFill(col, 0.35);
    surface.drawCircle(0, 0, R * 0.97);
    surface.endFill();
    holder.addChild(surface);
    t.surface = surface;

    // Второй слой поверхности (внутренний, темнее к центру воронки).
    const inner = new PIXI.Graphics();
    inner.beginFill(0x000000, 0.35);
    inner.drawCircle(0, 0, R * 0.45);
    inner.endFill();
    holder.addChild(inner);
    t.inner = inner;

    // 4) Настоящая спираль-воронка: непрерывная линия, закручивающаяся к центру.
    const spiral = new PIXI.Graphics();
    spiral.lineStyle(R * 0.045, 0xffffff, 0.85);
    buildSpiral(spiral, R);
    spiral.blendMode = PIXI.BLEND_MODES.ADD;
    holder.addChild(spiral);
    t.spiral = spiral;

    // 5) Переливающийся "огонь" по кругу: несколько дуг-бэндов цвета, вращаются.
    const flames = [];
    for (let i = 0; i < 3; i++) {
        const f = new PIXI.Graphics();
        f.lineStyle(R * 0.06, col, 0.5);
        f.arc(0, 0, R * 0.78, 0, Math.PI * 1.2);
        f.rotation = (Math.PI * 2 / 3) * i;
        f.blendMode = PIXI.BLEND_MODES.ADD;
        holder.addChild(f);
        flames.push(f);
    }
    t.flames = flames;

    // 6) Мерцающие частицы-искры, летающие по орбитам разного радиуса.
    const particles = [];
    const n = 10;
    for (let i = 0; i < n; i++) {
        const dot = new PIXI.Graphics();
        dot.beginFill(0xffffff, 0.95);
        dot.drawCircle(0, 0, R * (0.04 + Math.random() * 0.04));
        dot.endFill();
        dot.blendMode = PIXI.BLEND_MODES.ADD;
        holder.addChild(dot);
        particles.push({
            sprite: dot,
            angle: (Math.PI * 2 / n) * i,
            radius: R * (0.15 + Math.random() * 0.6),
            speed: (0.6 + Math.random() * 1.3),
            dir: Math.random() < 0.5 ? 1 : -1,
        });
    }
    t.particles = particles;

    holder.position.set(p.x, p.y);
    world.addChild(holder);
    return holder;
}

// buildSpiral рисует непрерывную спираль, закручивающуюся от края к центру.
function buildSpiral(g, R) {
    const turns = 3.2; // сколько оборотов
    const steps = 90;
    let first = true;
    for (let i = 0; i <= steps; i++) {
        const tt = i / steps;
        const rad = R * 0.96 * (1 - tt);       // радиус уменьшается к центру
        const ang = tt * turns * Math.PI * 2;  // угол нарастает
        const x = Math.cos(ang) * rad;
        const y = Math.sin(ang) * rad;
        if (first) { g.moveTo(x, y); first = false; }
        else g.lineTo(x, y);
    }
}

// drawPlayerSprite — красивый шар игрока: внешнее свечение, основной круг,
// блик сверху, тёмный обод снизу, светлая окантовка и кольцо команды.
function drawPlayerSprite(g, color, size, me, team) {
    g.clear();
    // Внешнее мягкое свечение.
    g.beginFill(color, 0.22);
    g.drawCircle(0, 0, size + 6);
    g.endFill();
    // Основной круг.
    g.beginFill(color);
    g.drawCircle(0, 0, size);
    g.endFill();
    // Блик сверху-слева (светлее).
    const light = petalShade(color, 1.5);
    g.beginFill(light, 0.55);
    g.drawEllipse(-size * 0.28, -size * 0.38, size * 0.52, size * 0.32);
    g.endFill();
    // Тёмный обод снизу (объём).
    const dark = petalShade(color, 0.6);
    g.lineStyle(2, dark, 0.9);
    g.drawCircle(0, 0, size);
    // Светлая окантовка (у себя — яркая белая).
    g.lineStyle(me ? 3 : 1.5, me ? 0xffffff : petalShade(color, 1.7), 0.95);
    g.drawCircle(0, 0, size + 1);
    // Кольцо команды.
    if (team !== 0) {
        g.lineStyle(2, team === 1 ? 0xe74c3c : 0x9b59b6, 0.9);
        g.drawCircle(0, 0, size + 4);
    }
    g.lineStyle(0);
}

// ----- Отрисовка игроков (круги), мобов, петалей и HP -----
function render(state) {
    const players = state.players || [];
    const seen = new Set();

    // Волновой режим: team-профили слева + Wave HUD (бар/ростер).
    if (waveModeClient) {
        updateTeamPanel(players);
        updateWaveHud(state);
    }

    for (const p of players) {
        seen.add(p.id);

        let actor = actors[p.id];
        if (!actor) {
            const isMe = p.id === myId;
            const sprite = new PIXI.Graphics();
            drawPlayerSprite(sprite, p.color, p.size, isMe, p.team || 0);
            world.addChild(sprite);

            const text = new PIXI.Text(p.name, {
                fontFamily: 'Arial', fontSize: 14, fill: '#ffffff',
                stroke: '#000000', strokeThickness: 3,
            });
            text.anchor.set(0.5);
            sprite.addChild(text);
            text.y = -p.size - 8;

            // Уровень цифрой возле ника, в цвет шара игрока.
            const lvl = new PIXI.Text(String(p.level || 0), {
                fontFamily: 'Arial', fontSize: 12, fontWeight: 'bold',
                fill: '#' + (p.color || 0xffffff).toString(16).padStart(6, '0'),
                stroke: '#000000', strokeThickness: 2,
            });
            lvl.anchor.set(0, 0.5);
            sprite.addChild(lvl);
            lvl.x = text.width / 2 + 5;
            lvl.y = text.y;

            sprite.position.set(p.x, p.y);
            // Оверлей хитбокса игрока (включается в настройках).
            const hitG = new PIXI.Graphics();
            sprite.addChildAt(hitG, 0);
            // 3D-кольцо Saturn — вокруг КАЖДОГО игрока с живой saturn-петалью
            // (кто её носит, у того над планетой висит кольцо, как у Сатурна).
            const satRing = new PIXI.Graphics();
            satRing.visible = false;
            sprite.addChildAt(satRing, 0);
            // Плавное появление игрока (фейд-ин за ~200 мс).
            sprite.alpha = 0;

            actor = { sprite, me: isMe, targetX: p.x, targetY: p.y, drawnSize: p.size, nameText: text, levelText: lvl, team: p.team || 0, hitG, satRing };
            actors[p.id] = actor;
        }
        // Фейд-ин после появления.
        if (actor.sprite.alpha < 1) actor.sprite.alpha = Math.min(1, actor.sprite.alpha + 0.14);
        // Хитбокс игрока: круг радиусом p.size.
        if (actor.hitG) {
            actor.hitG.clear();
            if (showHitPlayers) {
                actor.hitG.lineStyle(1.5, 0x2ecc71, 0.8);
                actor.hitG.drawCircle(0, 0, p.size);
                actor.hitG.lineStyle(0);
            }
        }

        // Кольцо-аура Saturn (3D-кольцо как у Сатурна) — рисуется У ВСЕХ
        // игроков, у кого на орбите ЖИВАЯ saturn-петаль (иначе бы «нагишом»
        // у соседей висела зона урона). Живость заполняет renderPetals
        // в saturnAlive[id] = редкость. Сам петаль Saturn на орбите НЕ
        // рисуем — он «не петаль», а кольцо вокруг планеты.
        if (actor.satRing && saturnAlive[p.id] != null) {
            const satRar = saturnAlive[p.id];
            const ringR = SAT_RING_BASE + SAT_RING_PER_RAR * satRar;
            const tt = performance.now() / 1000;
            const g = actor.satRing;
            g.visible = true;
            g.clear();
            // «Планета» = сам игрок. Кольцо — сплющенный эллипс (вид под углом):
            // задняя половина темнее, передняя ярче и толще — эффект 3D.
            const rx = ringR;
            const ry = ringR * 0.38;
            // Едва заметная заливка всей зоны урона.
            g.beginFill(0x73c6b6, 0.05);
            g.drawCircle(0, 0, ringR);
            g.endFill();
            // Задняя половина (верх) — темнее.
            g.lineStyle(4.5, 0x4d9486, 0.55);
            g.drawEllipse(0, 0, rx, ry);
            // Внутренняя тонкая кромка.
            g.lineStyle(1.4, 0x73c6b6, 0.5);
            g.drawEllipse(0, 0, rx * 0.86, ry * 0.86);
            // Передняя половина (низ, «ближе к камере») — яркая.
            g.lineStyle(7, 0xd8f5ee, 0.8);
            g.moveTo(-rx, 0);
            g.bezierCurveTo(-rx, ry * 0.55, -rx * 0.55, ry, 0, ry);
            g.bezierCurveTo(rx * 0.55, ry, rx, ry * 0.55, rx, 0);
            g.lineStyle(0);
            // Луны, бегущие по кольцу (по эллипсу).
            for (let i = 0; i < 3; i++) {
                const ang = tt * 1.1 + i * 2.094;
                g.beginFill(0xd8f5ee, 0.9);
                g.drawCircle(Math.cos(ang) * rx, Math.sin(ang) * ry, 2.4 + (i % 2));
                g.endFill();
            }
        } else if (actor.satRing && actor.satRing.visible) {
            actor.satRing.visible = false;
            actor.satRing.clear();
        }

        actor.targetX = p.x;
        actor.targetY = p.y;
        if (p.id === myId) { myPlayerX = p.x; myPlayerY = p.y; }

        // Уровень: цифра обновляется в реальном времени, цвет = макс. редкость
        // петалей ТОЛЬКО ТЕКУЩЕГО АКТИВНОГО хотбара (пустой — цвет Point).
        // Для других игроков сервер шлёт готовый maxHotRar (без полных хотбаров,
        // чтобы резать трафик); для себя хотбары в сообщении есть — считаем как раньше.
        if (actor.levelText) {
            const lvlStr = String(p.level || 0);
            if (actor.levelText.text !== lvlStr) actor.levelText.text = lvlStr;
            let maxRar = p.maxHotRar || 0;
            const ab = (p.activeBar === 1) ? 1 : 0;
            const actBar = Array.isArray(p.hotbars) ? p.hotbars[ab] : null;
            if (Array.isArray(actBar)) {
                for (const def of actBar) {
                    if (def && (def.rarity || 0) > maxRar) maxRar = def.rarity || 0;
                }
            }
            actor.levelText.style.fill = rarityColors[Math.min(maxRar, rarityColors.length - 1)];
            if (actor.nameText) {
                actor.levelText.x = actor.nameText.width / 2 + 5;
            }
        }

        // Перерисовываем круг/ник, если размер игрока или команда изменились.
        if (p.size != null && p.size !== actor.drawnSize || (p.team || 0) !== actor.team) {
            drawPlayerSprite(actor.sprite, p.color, p.size, actor.me, p.team || 0);
            if (actor.nameText) actor.nameText.y = -p.size - 8;
            if (actor.levelText) { actor.levelText.x = (actor.nameText ? actor.nameText.width / 2 : 0) + 5; actor.levelText.y = -p.size - 8; }
            actor.drawnSize = p.size;
            actor.team = p.team || 0;
        }

        // Пишем только своё HP.
        if (p.id === myId) {
            updateHP(p.hp, p.maxHp, p.shield, p.maxShield);
            myZone = p.zone;
            zoneLabel.textContent = `You are in ${zones[p.zone] ? zones[p.zone].name : '???'} Zone`;
            updateXP(p.level || 0, p.xp || 0, p.xpMax || 1);
            // Инвентарь игрока (петали-предметы). Обновляем меню при необходимости.
            if (p.inventory) myInventory = p.inventory;
            if (p.hotbars) myHotbars = p.hotbars;
            if (p.activeBar != null) myActiveBar = p.activeBar;
            updateMenuLevel();
            // Внутриигровая панель инвентаря: перерисовка при изменении.
            if (gameInvOpen) updateGameInvPanel(false);
        }
    }

    // Убираем вышедших игроков, но НЕ себя (с плавным растворением).
    for (const id of Object.keys(actors)) {
        if (!seen.has(id) && id !== myId) {
            fadeOutThenRemove(actors[id].sprite);
            delete actors[id];
        }
    }

    renderMobs(state.mobs || []);
    renderPetals(state.petals || []);
    renderGround(state.ground || []);
    renderProjectiles(state.projectiles || []);
    renderChains(state.chains || []);
}

function updateHP(hp, maxHp, shield, maxShield) {
    lastMaxHP = maxHp;
    const nickEl = document.getElementById('playerNick');
    if (nickEl && nickEl.textContent !== myNickname) nickEl.textContent = myNickname || '-';
    const pct = Math.max(0, Math.min(100, hp / maxHp * 100));
    hpFill.style.width = pct + '%';

    // Синяя полоска щита (броня перед HP). Скрыта, если щита нет.
    const shPct = Math.max(0, Math.min(100, (shield || 0) / (maxShield || 1) * 100));
    if (maxShield > 0) {
        shieldBar.style.display = 'block';
        shieldFill.style.width = shPct + '%';
    } else {
        shieldBar.style.display = 'none';
    }

    // Красивый градиент, зависящий от здоровья: зелёный -> жёлтый -> красный.
    let grad, glow;
    if (pct > 60) {
        grad = 'linear-gradient(90deg, rgba(0,0,0,0.55) 0%, rgba(0,0,0,0) 26%, #7be0ff 55%, #2fbf71 100%)';
        glow = 'rgba(47,191,113,0.55)';
    } else if (pct > 30) {
        grad = 'linear-gradient(90deg, rgba(0,0,0,0.55) 0%, rgba(0,0,0,0) 26%, #ffe27a 50%, #f39c12 100%)';
        glow = 'rgba(243,156,18,0.6)';
    } else {
        grad = 'linear-gradient(90deg, rgba(0,0,0,0.55) 0%, rgba(0,0,0,0) 26%, #ff8a6b 45%, #e74c3c 100%)';
        glow = 'rgba(231,76,60,0.65)';
    }
    hpFill.style.background = grad;
    hpFill.style.boxShadow = `inset 0 2px 6px rgba(255,255,255,0.35), inset 0 -4px 8px rgba(0,0,0,0.35), 0 0 14px ${glow}`;

    hpText.textContent = `${Math.ceil(hp)} / ${Math.ceil(maxHp)}`;

    // При низком здоровье смена цвета HP-текста.
    hpText.style.color = pct <= 25 ? '#ff9a8c' : '#ffffff';

    // Лёгкая вибрация полоски, когда здоровья мало (тревожность).
    if (pct > 0 && pct <= 20) {
        hpBar.style.animation = 'hpDanger 0.5s infinite';
    } else {
        hpBar.style.animation = 'none';
    }
}

// ----- Шкала опыта -----
function updateXP(level, xp, xpMax) {
    myLevel = level;
    myXP = xp;
    myXPMax = xpMax || 1;
    const pct = Math.max(0, Math.min(100, xp / (xpMax || 1) * 100));
    xpFill.style.width = pct + '%';
    xpText.textContent = `Lv. ${level} — ${Math.floor(xp)} / ${Math.floor(myXPMax)} XP`;
    updateMenuLevel();
}
// ----- Отрисовка мобов -----
function renderMobs(mobs) {
    const seen = new Set();
    const bosses = [];
    for (const m of mobs) {
        seen.add(m.id);
        if (!m.ally && (m.rarity === 'Chaos' || m.rarity === 'Fractal' || m.rarity === 'Hexagon')) bosses.push(m);
        let a = mobActors[m.id];
        if (!a) {
            const c = new PIXI.Container();
            const g = new PIXI.Graphics();
            drawMobShape(g, m);
            c.addChild(g);

            // Специальная аура для Fractal/Chaos (боссы) и Hexagon (мини-боссы):
            // светящееся кольцо + орбитальные осколки за формой моба (рисунок
            // обновляется каждый кадр в tick()).
            if (!m.ally && (m.rarityI || 0) >= 5) {
                const aura = new PIXI.Graphics();
                c.addChildAt(aura, 0);
                a = { aura };
            } else {
                a = {};
            }

            // Миньон (союзный моб Entropy): зелёное пульсирующее кольцо-сигнатура,
            // чтобы отличить его от вражеских мобов той же формы/цвета.
            if (m.ally) {
                const ring = new PIXI.Graphics();
                c.addChildAt(ring, 0);
                a.ring = ring;
            }

            // Аура яда (моб отравлен петалем-«гадюкой»): пульсирующее зелёное
            // кольцо + «пузырьки» — рисуется каждый кадр, когда m.poison.
            const pRing = new PIXI.Graphics();
            c.addChildAt(pRing, 0);
            a.pRing = pRing;

            // Аура замедления (петель rhombus): морозно-синее кольцо с «шестерёнкой»
            // из тиков по окружности — рисуется, когда m.slow.
            const sRing = new PIXI.Graphics();
            c.addChildAt(sRing, 0);
            a.sRing = sRing;

            // Имя + редкость — единый лейбл-блок, отцентрованный над мобом (ровно,
            // не зависит от длины имени/размера моба). Тёмная «стеклянная»
            // таблетка имени с глянцем и обводкой цвета редкости, рядом —
            // светящийся цветной бейдж редкости.
            // БЕЗ dropShadow у имени: тень раздувала измеренную ширину текста и
            // таблетка выглядела «съехавшей» относительно букв.
            const nar = m.rarityI != null ? rarityNum(m.rarityI) : 0x9aa0a6;
            const labelRow = new PIXI.Container();

            // Компактность лейбла зависит от размера моба: у мелких
            // (Point..Square) имя и чип редкости меньше и плотнее — иначе под
            // крошечным спрайтом висел непропорционально длинный «столбик»
            // из полоски HP + имени + бейджа с воздухом между ними.
            const compact = Math.max(0, Math.min(1, (m.size - 16) / 18)); // 0 супер-мелкий, 1 средний+
            const nameFs = Math.round(9 + compact * 2); // размер имени 9..11
            const badgeFs = Math.round(7 + compact * 2); // размер чипа 7..9
            const nhPad = 4 + compact * 3; // боковой паддинг таблетки 4..7
            const nhH = 3 + compact * 1; // вертикальный паддинг таблетки 3..4
            const gap = 7 + compact * 6; // зазор таблетка|чип 7..13
            const bpad = 5 + compact * 1; // боковой паддинг чипа 5..6
            const bpadV = 2 + compact * 0.5; // верт. паддинг чипа 2..2.5

            // --- Имя: стеклянная таблетка с глянцем и кольцом цвета редкости.
            const nameHolder = new PIXI.Container();
            const nameText = new PIXI.Text(m.name || '?', {
                fontFamily: 'Arial, sans-serif', fontSize: nameFs, fontWeight: 'bold', fill: '#ffffff',
                stroke: '#000000', strokeThickness: 2.5,
            });
            nameText.anchor.set(0.5, 0.5);
            const pillW = nameText.width + nhPad * 2;
            const pillH = nameText.height + nhH;
            const namePill = new PIXI.Graphics();
            // мягкий ореол цвета редкости позади таблетки
            namePill.beginFill(nar, 0.16);
            namePill.drawRoundedRect(-pillW / 2 - 2, -pillH / 2 - 2, pillW + 4, pillH + 4, pillH / 2 + 2);
            namePill.endFill();
            // корпус
            namePill.beginFill(0x111920, 0.74);
            namePill.drawRoundedRect(-pillW / 2, -pillH / 2, pillW, pillH, pillH / 2);
            namePill.endFill();
            // глянцевый верх
            namePill.beginFill(0xffffff, 0.10);
            namePill.drawRoundedRect(-pillW / 2 + 1.5, -pillH / 2 + 1.5, pillW - 3, pillH / 2 - 2, Math.max(0.5, pillH / 2 - 2));
            namePill.endFill();
            // кольцо цвета редкости
            namePill.lineStyle(1, nar, 0.55);
            namePill.drawRoundedRect(-pillW / 2, -pillH / 2, pillW, pillH, pillH / 2);
            namePill.lineStyle(0);
            nameHolder.addChild(namePill, nameText);

            // --- Редкость: цветной светящийся «чип» с тёмным текстом.
            const badge = new PIXI.Container();
            const bt = new PIXI.Text(m.rarity || '', {
                fontFamily: 'Arial, sans-serif', fontSize: badgeFs, fontWeight: 'bold', fill: '#121821',
            });
            bt.anchor.set(0.5, 0.5);
            const bW = bt.width + bpad * 2;
            const bH = bt.height + bpadV * 2;
            const bgPill = new PIXI.Graphics();
            bgPill.beginFill(nar, 0.28);
            bgPill.drawRoundedRect(-bW / 2 - 2.5, -bH / 2 - 2.5, bW + 5, bH + 5, (bH + 5) / 2);
            bgPill.endFill();
            bgPill.beginFill(nar, 0.95);
            bgPill.drawRoundedRect(-bW / 2, -bH / 2, bW, bH, bH / 2);
            bgPill.endFill();
            bgPill.lineStyle(1, 0xffffff, 0.6);
            bgPill.drawRoundedRect(-bW / 2, -bH / 2, bW, bH, bH / 2);
            bgPill.lineStyle(0);
            badge.addChild(bgPill, bt);

            // Раскладываем весь блок симметрично вокруг центра моба (x=0):
            // таблетка | зазор gap | чип.
            const totalW = pillW + gap + bW;
            nameHolder.position.set(-totalW / 2 + pillW / 2, 0);
            badge.position.set(-totalW / 2 + pillW + gap + bW / 2, 0);
            labelRow.addChild(nameHolder, badge);
            // Полоска HP вплотную под мобом, лейбл сразу под ней — без лишнего
            // воздуха, но имя/редкость всё равно никогда не наезжают на HП.
            const barTop = m.size + 2;       // верх полоски HP
            const bhBar = 6;                 // высота полоски HP (тот же bh ниже)
            const rowH = Math.max(pillH, bH);
            labelRow.position.set(0, barTop + bhBar + 2 + rowH / 2);
            c.addChild(labelRow);

            // Витринный мокап TestRoom: крупный номер над мобом в тёмном чипе
            // с бирюзовым кольцом (заметно поверх любой геометрии дизайна).
            if (m.showcase && m.label) {
                const shLabel = new PIXI.Text(m.label, {
                    fontFamily: 'Arial, sans-serif', fontSize: 30, fontWeight: 'bold', fill: '#eafffb',
                    stroke: '#0b3d33', strokeThickness: 6,
                });
                shLabel.anchor.set(0.5, 0.5);
                const shChip = new PIXI.Graphics();
                shChip.beginFill(0x0b2430, 0.86);
                shChip.lineStyle(2.5, 0x1abc9c, 0.95);
                shChip.drawCircle(0, 0, 26);
                shChip.endFill();
                shChip.lineStyle(0);
                const shBox = new PIXI.Container();
                shBox.addChild(shChip, shLabel);
                shBox.position.set(0, -(m.size + 28));
                c.addChild(shBox);
                a.showLabel = shBox;
            }

            // Полоска HP: подложка-тень для объёма, фон, светлый след урона и заполнитель.
            // Ширина пропорциональна мобу (у мелких чуть уже спрайта, как и сам лейбл).
            const bw = Math.max(26, m.size * 1.8);
            const bh = 6;
            const barY = barTop; // слева под мобом, имя/редкость ниже полоски

            // Подложка-тень + затемнённый фон полоски.
            const barBg = new PIXI.Graphics();
            barBg.beginFill(0x000000, 0.35);
            barBg.drawRoundedRect(1, 2, bw + 2, bh + 2, (bh + 2) / 2);
            barBg.endFill();
            barBg.beginFill(0x0a0f15, 1);
            barBg.lineStyle(1.2, 0x000000, 1);
            barBg.drawRoundedRect(0, 0, bw, bh, bh / 2);
            barBg.endFill();
            barBg.position.set(-bw / 2, barY);
            c.addChild(barBg);

            // Светлый след недавнего урона — плавно тает к текущему HP в tick().
            const barGhost = new PIXI.Graphics();
            barGhost.position.set(-bw / 2, barY);
            c.addChild(barGhost);

            // Основной заполнитель (рисуется в redrawMobHP).
            const barFill = new PIXI.Graphics();
            barFill.position.set(-bw / 2, barY);
            c.addChild(barFill);

            // Глянцевая «шапка» сверху заполнителя — придаёт объём.
            const barShine = new PIXI.Graphics();
            barShine.position.set(-bw / 2, barY);
            c.addChild(barShine);

            // Светлая окантовка поверх всего бара — чёткий «стеклянный» контур.
            const barEdge = new PIXI.Graphics();
            barEdge.position.set(-bw / 2, barY);
            c.addChild(barEdge);

            c.position.set(m.x, m.y);
            world.addChild(c);
            a.c = c;
            // Оверлей хитбокса моба (включается в настройках). Кладём ПОВЕРХ
            // спрайта: у круглых мобов (dot → shape circle) непрозрачный спрайт
            // ровно радиуса size полностью закрывал кольцо, если хитбокс позади.
            const hitG = new PIXI.Graphics();
            c.addChild(hitG);
            c.setChildIndex(hitG, c.getChildIndex(g) + 1);
            a.hitG = hitG;
            // Плавное появление: новый моб влетает прозрачностью.
            c.alpha = 0;
            a.g = g;
            a.barBg = barBg; a.barGhost = barGhost; a.barFill = barFill; a.barShine = barShine; a.barEdge = barEdge;
            a.labelRow = labelRow; a.targetX = m.x; a.targetY = m.y; a.size = m.size;
            a.bw = bw; a.bh = bh; a.fillW = 0; a.ghost = 0; a.rot = 0;
            a.rarityI = m.rarityI || 0;
            // Мелкие мобы (Point/Line-редкость или крошечные) не таскают за собой
            // «столбик» из HP-бара и имени под каждым: кластер скрыт и всплывает
            // только когда моба ударили (см. updateMobActors, a.tinyInfo).
            a.infoParts = [barBg, barGhost, barFill, barShine, barEdge, labelRow];
            // ТЕСТ: скрытие HP+имени ДО первого удара для ВСЕХ мобов
            // (кроме витринных мокапов TestRoom и союзных миньонов).
            a.tinyInfo = !a.showcase && !m.ally && !(m.bosspetals || []).length;
            a.infoAlpha = a.tinyInfo ? 0 : 1;
            if (a.tinyInfo) {
                for (const p of a.infoParts) p.alpha = 0;
            }
            mobActors[m.id] = a;
        }
        // Фейд-ин: догоняем прозрачность до 1 после появления.
        if (a.c.alpha < 1) a.c.alpha = Math.min(1, a.c.alpha + 0.14);
        a.owner = m.owner || '';
        a.ally = !!m.ally;
        // Хитбокс моба: круг радиусом ровно m.size (коллизия сервера).
        if (a.hitG) {
            a.hitG.clear();
            if (showHitMobs) {
                a.hitG.lineStyle(1.5, 0xff5555, 0.75);
                a.hitG.drawCircle(0, 0, m.size);
                a.hitG.lineStyle(0);
            }
        }
        a.targetX = m.x;
        a.targetY = m.y;
        a.size = m.size;
        if (a.showLabel) a.showLabel.y = -(m.size + 28);
        a.shape = m.shape;
        a.color = m.color;
        a.variant = m.variant || '';
        a.type = m.type;
        a.showcase = !!m.showcase;
        a.label = m.label || '';
        a.bossPetals = m.bosspetals || [];
        const rawRot = m.speed > 0 ? Math.atan2(m.dirY || 0, m.dirX || 0) : 0;
        if (m.shape === 'wall') {
            // FragWall — стена: направление только по осям, поворот плавный.
            const q = Math.PI / 2;
            a.rot = Math.round(rawRot / q) * q;
            a.slowTurn = true;
        } else {
            a.rot = rawRot;
            a.slowTurn = false;
        }
        a.g.clear();
        drawMobShape(a.g, m);
        // Beam: «энергетический круг», из которого исходят шары — пульсирующее
        // кольцо постоянного действия.
        if (a.type === 'beam') {
            const t = performance.now() / 1000;
            const R = m.size * (1.5 + 0.12 * Math.sin(t * 3.2));
            a.g.lineStyle(2.5, m.color || 0xffd32e, 0.5 + 0.2 * Math.sin(t * 3.2));
            a.g.drawCircle(0, 0, R);
            a.g.lineStyle(1, 0xffffff, 0.3 * (0.5 + 0.5 * Math.sin(t * 2.1)));
            a.g.drawCircle(0, 0, R * 0.82);
            a.g.lineStyle(0);
        }
        // Союзное кольцо миньона — пульсирует каждый кадр.
        if (a.ring) {
            const pulse = 3 + 0.5 * Math.sin(performance.now() / 260);
            a.ring.clear();
            a.ring.lineStyle(1.6, 0x66ffdd, 0.55);
            a.ring.drawCircle(0, 0, a.size * 1.5 + pulse);
        }
        // Аура яда: пульсирующее кольцо и «пузырьки», если моб отравлен.
        if (a.pRing) {
            a.pRing.clear();
            if (m.poison) {
                const t = performance.now() / 900;
                a.pRing.lineStyle(1.8, 0x39ff5e, 0.8);
                a.pRing.beginFill(0x4dff66, 0.10);
                a.pRing.drawCircle(0, 0, a.size * 1.35 + Math.sin(t * 2.2) * 1.6);
                a.pRing.endFill();
                a.pRing.lineStyle(1.2, 0x7dff9b, 0.85);
                for (let i = 0; i < 3; i++) {
                    const ang = t * 1.5 + i * 2.094;
                    const rad = a.size * (1.45 + 0.28 * Math.sin(t * 3 + i * 2));
                    a.pRing.beginFill(0xb6ffc4, 0.9);
                    a.pRing.drawCircle(Math.cos(ang) * rad, Math.sin(ang) * rad, 1.4 + (i % 2) * 0.7);
                    a.pRing.endFill();
                }
            }
        }
        // Аура замедления: морозно-синее пульсирующее кольцо с «тикающими»
        // зубцами по окружности, если моб замедлен (rhombus).
        if (a.sRing) {
            a.sRing.clear();
            if (m.slow) {
                const tS = performance.now() / 800;
                a.sRing.lineStyle(1.8, 0x4dc3ff, 0.8);
                a.sRing.beginFill(0x54a8ff, 0.10);
                a.sRing.drawCircle(0, 0, a.size * 1.35 + Math.sin(tS * 2.0) * 1.6);
                a.sRing.endFill();
                const teeth = 10;
                a.sRing.lineStyle(1.4, 0x9fdcff, 0.9);
                for (let i = 0; i < teeth; i++) {
                    const ang = (i / teeth) * Math.PI * 2;
                    const r1 = a.size * 1.42;
                    const r2 = a.size * (1.56 + 0.1 * Math.sin(tS * 3 + i));
                    a.sRing.moveTo(Math.cos(ang) * r1, Math.sin(ang) * r1);
                    a.sRing.lineTo(Math.cos(ang) * r2, Math.sin(ang) * r2);
                }
                a.sRing.lineStyle(0);
            }
        }
        redrawMobHP(a, m);
        // Мелкие мобы: блок HP+имя всплывает только когда моб подранен
        // (hp < макс — без регена он не вернётся, поэтому бар не гаснет) или
        // прямо сейчас получил удар (светлый след «эха» ещё тает). Ровно это
        // убирает визуальный мусор под каждом Point-мобом на поле.
        if (a.tinyInfo) {
            const want = (m.hp < m.maxHp - 0.01 || a.ghost > a.fillW) ? 1 : 0;
            a.infoAlpha += (want - a.infoAlpha) * 0.25;
            if (want === 0 && a.infoAlpha < 0.02) a.infoAlpha = 0;
            for (const p of a.infoParts) p.alpha = a.infoAlpha;
        }
    }
    for (const id of Object.keys(mobActors)) {
        if (!seen.has(id)) {
            fadeOutThenRemove(mobActors[id].c);
            delete mobActors[id];
        }
    }
    updateBossHUD(bosses);
}

// Обновляет босс-бары (до 4) для Chaos и Fractal мобов, сортирует по HP.
function updateBossHUD(bosses) {
    const hud = document.getElementById('bossHud');
    if (!hud) return;
    if (!bosses || bosses.length === 0) {
        hud.classList.remove('show');
        return;
    }
    // Крупнейшие боссы первыми.
    bosses.sort((a, b) => (b.maxHp || 0) - (a.maxHp || 0));
    const entries = hud.querySelectorAll('.boss-entry');
    for (let i = 0; i < entries.length; i++) {
        const b = bosses[i];
        if (!b) {
            entries[i].style.display = 'none';
            continue;
        }
        entries[i].style.display = '';
        const btype = b.rarity === 'Chaos' ? 'chaos' : (b.rarity === 'Fractal' ? 'fractal' : (b.rarity === 'Hexagon' ? 'hexagon' : ''));
        entries[i].classList.toggle('chaos', btype === 'chaos');
        entries[i].classList.toggle('fractal', btype === 'fractal');
        const nameEl = entries[i].querySelector('.boss-name');
        if (nameEl && nameEl.textContent !== (b.name || b.type || '')) nameEl.textContent = b.name || b.type || '';
        const rarityEl = entries[i].querySelector('.boss-rarity');
        if (rarityEl && rarityEl.textContent !== (b.rarity || '')) rarityEl.textContent = b.rarity || '';
        const pct = Math.max(0, Math.min(100, b.hp / b.maxHp * 100));
        const fill = entries[i].querySelector('.boss-fill');
        if (fill) {
            fill.style.width = pct + '%';
            fill.classList.toggle('low', pct < 30);
            if (b.rarityI != null) {
                const c = rarityHex(b.rarityI);
                fill.style.background = `rgba(${(c >> 16) & 255}, ${(c >> 8) & 255}, ${c & 255}, 0.85)`;
            }
        }
        const txt = entries[i].querySelector('.boss-text');
        if (txt) txt.textContent = `${Math.ceil(b.hp)} / ${Math.ceil(b.maxHp)}`;
    }
    hud.classList.add('show');
}

// Орбитальные петали боссов: Fractal — 4, Chaos — 6, у каждого свой график с
// формой типа моба (см. tick мобов). Здесь рисуется только пульсирующий ореол
// редкости вокруг тела босса (Fractal фиолетовый, Chaos малиновый).
function drawMobAura(g, m, t) {
    g.clear();
    const r = m.size || 1;
    if (m.rarityI >= 7) {
        const glow = 0xff2d95;
        const p = 0.5 + 0.5 * Math.sin(t * 2.2);
        g.beginFill(0x000000, 0.30);
        g.drawCircle(0, 0, r * (1.45 + p * 0.25));
        g.endFill();
        g.lineStyle(3 + p * 2.5, glow, 0.35 + 0.35 * p);
        g.drawCircle(0, 0, r * (1.28 + p * 0.16));
        g.lineStyle(1.5, 0xffffff, 0.15 + 0.2 * p);
        g.drawCircle(0, 0, r * (1.52 + p * 0.2));
    } else if (m.rarityI === 6) {
        const glow = 0x9b59b6;
        const p = 0.5 + 0.5 * Math.sin(t * 1.7);
        g.beginFill(0x1a0033, 0.35);
        g.drawCircle(0, 0, r * (1.22 + p * 0.18));
        g.endFill();
        g.lineStyle(3 + p * 1.5, glow, 0.55 + 0.25 * p);
        g.drawCircle(0, 0, r * (1.14 + p * 0.1));
        g.lineStyle(1.2, 0xd6a4ff, 0.4 + 0.3 * p);
        g.drawCircle(0, 0, r * (1.34 + p * 0.14));
    } else if (m.rarityI === 5) {
        // Mini-босс Hexagon: янтарная «шестиугольная оправа» — вращающийся
        // внешний гекс, противовращающийся внутренний и глиттеры на вершинах.
        const glow = 0xffc93c;
        const p = 0.5 + 0.5 * Math.sin(t * 1.9);
        // Мягкая тёмная подложка (не круглая аура, а сам гексо-силуэт).
        g.beginFill(0x231a06, 0.32);
        hexPath(g, r * 1.3 + p * r * 0.05, Math.PI / 6);
        g.endFill();
        // Внешний вращающийся гексо-контур.
        g.lineStyle(2.6 + p * 1.6, glow, 0.6 + 0.3 * p);
        hexPath(g, r * 1.18 + p * r * 0.07, t * 0.7);
        // Внутренний противовращающийся тонкий гекс.
        g.lineStyle(1.4, 0xfff3c4, 0.45 + 0.3 * p);
        hexPath(g, r * 1.02, -t * 0.5);
        g.lineStyle(0);
        // Глиттеры на вершинах внешнего гекса.
        for (let i = 0; i < 6; i++) {
            const a = t * 0.7 + i * Math.PI / 3;
            const gr = r * 1.18 + p * r * 0.07;
            g.beginFill(0xfff6d8, 0.85);
            g.drawCircle(Math.cos(a) * gr, Math.sin(a) * gr, 2.2 + p);
            g.endFill();
        }
    }
}

// Рисует шестиугольник с центром в (0,0) и вершинами на радиусе r.
function hexPath(g, r, phase) {
    g.moveTo(Math.cos(phase) * r, Math.sin(phase) * r);
    for (let i = 1; i <= 6; i++) {
        const a = phase + i * Math.PI / 3;
        g.lineTo(Math.cos(a) * r, Math.sin(a) * r);
    }
    g.closePath();
}

// Рисует звезду с центром в (0,0): n лучей, внутренний радиус r2, внешний r1.
function starPath(g, n, r1, r2, phase) {
    for (let i = 0; i < n * 2; i++) {
        const rr = (i % 2 === 0) ? r1 : r2;
        const a = phase + i * Math.PI / n;
        if (i === 0) g.moveTo(Math.cos(a) * rr, Math.sin(a) * rr);
        else g.lineTo(Math.cos(a) * rr, Math.sin(a) * rr);
    }
    g.closePath();
}

// Вспомогательный многоугольник-веер (дуга из n точек).
function arcPath(g, r, a0, a1, steps) {
    for (let i = 0; i <= steps; i++) {
        const a = a0 + (a1 - a0) * i / steps;
        if (i === 0) g.moveTo(Math.cos(a) * r, Math.sin(a) * r);
        else g.lineTo(Math.cos(a) * r, Math.sin(a) * r);
    }
}

// drawShowcaseDesign — 25 витринных мокапов мобов TestRoom (зона 16): 14 Comet,
// 15 Beam (следующие кандидаты для зоны «Line II») и ромбовый кластер 16-25
// (Diamond..Prism). Каждый дизайн уникален и рисуется на радиус size вокруг
// центра (0,0). Это не реальные мобы, а «концепты» для выбора: у каждого есть
// номер 1-25. Анимация (галактика, червяк, орбиты) берётся из
// performance.now(): акторы перерисовываются каждый кадр, поэтому время
// можно использовать прямо в рисовании.
// перерисовываются на каждом тике состояния, поэтому время «течёт».
function drawShowcaseDesign(g, idx, size, col, label) {
    // Витрина 2.0 (порт родителя): слоты 24-77 — авторские дизайны
    // SHOWCASE_DESIGNS, остальные — легаси-витрина (включая 20/21/26/39/68/70/111
    // — формы новых мобов Arrowhead/Shardwing/Kite-Lancer/Frame/Nova/Mirror/Trinity).
    let slot = parseInt(label, 10);
    if (!slot || slot < 1) slot = idx;
    if (typeof drawShowcaseSlot === 'function' && slot >= 24 && slot <= 77) {
        drawShowcaseSlot(slot, g, size, col);
        return;
    }
    if (typeof drawShowcaseLegacy === 'function') {
        drawShowcaseLegacy(g, idx, size, col);
        return;
    }
    g.beginFill(col, 0.9);
    g.drawCircle(0, 0, size * 0.8);
    g.endFill();
}

function drawMobShape(g, m) {
    g.clear();
    const size = m.size;
    const col = m.color != null ? m.color : 0x888888;
    const dark = shade(col, -60);
    const light = shade(col, 55);
    const shape = m.shape || 'circle';

    // Витринные мокапы TestRoom (shape "mockup_N") — также форма моба Prism.
    if (shape.indexOf('mockup_') === 0) {
        drawShowcaseDesign(g, parseInt(shape.slice(7), 10), size, col);
        return;
    }

    if (shape === 'square') {
        const s = size * 0.9;
        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(m.color, 1);
        g.drawRect(-s, -s, s * 2, s * 2);
        g.endFill();
        g.beginFill(0x000000, 0.28);
        g.drawRect(-s, -s * 0.05, s * 2, s * 0.95);
        g.endFill();
        g.lineStyle(3, light, 0.9);
        g.drawRect(-s * 0.62, -s * 0.62, s * 1.24, s * 1.24);
        g.lineStyle(4, light, 0.8);
        const c = s * 0.82;
        const l2 = s * 0.28;
        g.moveTo(-c, -c); g.lineTo(-c + l2, -c);
        g.moveTo(-c, -c); g.lineTo(-c, -c + l2);
        g.moveTo(c, -c); g.lineTo(c - l2, -c);
        g.moveTo(c, -c); g.lineTo(c, c - l2);
        g.moveTo(-c, c); g.lineTo(-c + l2, c);
        g.moveTo(-c, c); g.lineTo(-c, c - l2);
        g.moveTo(c, c); g.lineTo(c - l2, c);
        g.moveTo(c, c); g.lineTo(c, c - l2);
        g.beginFill(light, 0.9);
        g.drawCircle(0, 0, s * 0.18);
        g.endFill();
    } else if (shape === 'triangle') {
        const r = size * 1.25;
        // Треугольник «смотрит» вершиной вперёд (+x = направление движения).
        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(m.color, 1);
        g.moveTo(r, 0);
        g.lineTo(-r * 0.55, r * 0.85);
        g.lineTo(-r * 0.55, -r * 0.85);
        g.closePath();
        g.endFill();
        g.lineStyle(2, light, 0.9);
        g.beginFill(light, 0.25);
        g.moveTo(r * 0.45, 0);
        g.lineTo(-r * 0.2, r * 0.42);
        g.lineTo(-r * 0.2, -r * 0.42);
        g.closePath();
        g.endFill();
        g.beginFill(light, 0.95);
        g.drawCircle(0, 0, r * 0.10);
        g.endFill();
        // Очерк-стрелка из угла: продолжает вершину по ходу движения.
        g.lineStyle(3, light, 0.95);
        g.moveTo(r * 1.22, 0);
        g.lineTo(r * 0.74, r * 0.40);
        g.moveTo(r * 1.22, 0);
        g.lineTo(r * 0.74, -r * 0.40);
        g.lineStyle(0);
    } else if (shape === 'pentagon') {
        const r = size * 1.1;
        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(m.color, 1);
        g.moveTo(r * Math.cos(-Math.PI / 2), r * Math.sin(-Math.PI / 2));
        for (let i = 1; i <= 5; i++) {
            const a = -Math.PI / 2 + i * (2 * Math.PI / 5);
            g.lineTo(r * Math.cos(a), r * Math.sin(a));
        }
        g.endFill();
        g.beginFill(dark, 0.25);
        const ri = r * 0.55;
        g.moveTo(ri * Math.cos(-Math.PI / 2), ri * Math.sin(-Math.PI / 2));
        for (let i = 1; i <= 5; i++) {
            const a = -Math.PI / 2 + i * (2 * Math.PI / 5);
            g.lineTo(ri * Math.cos(a), ri * Math.sin(a));
        }
        g.endFill();
        g.lineStyle(2, light, 0.8);
        const ri2 = r * 0.4;
        g.moveTo(ri2 * Math.cos(-Math.PI / 2), ri2 * Math.sin(-Math.PI / 2));
        for (let i = 1; i <= 5; i++) {
            const a = -Math.PI / 2 + i * (2 * Math.PI / 5);
            g.lineTo(ri2 * Math.cos(a), ri2 * Math.sin(a));
        }
        g.beginFill(light, 0.9);
        g.drawCircle(0, 0, r * 0.1);
        g.endFill();
    } else if (shape === 'hexagon') {
        // Огранённый кристалл: свечение, фаcеты, спицы и светящаяся сердцевина.
        const r = size * 1.2;
        g.lineStyle(0);
        g.beginFill(col, 0.22);
        g.drawCircle(0, 0, r * 1.45);
        g.endFill();
        // Кристалл: шестиугольник с толстым контуром.
        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(m.color, 1);
        hexPath(g, r, 0);
        g.endFill();
        // Фасеты: светлая сторона сверху.
        g.beginFill(0xffffff, 0.14);
        hexPath(g, r, 0);
        g.endFill();
        // Тёмный обод снизу — визуальный объём.
        g.lineStyle(0);
        g.beginFill(dark, 0.35);
        g.moveTo(r, 0);
        for (let i = 0; i <= 3; i++) {
            const a = (i * Math.PI / 3) - Math.PI / 6;
            g.lineTo(Math.cos(a) * r, Math.sin(a) * r);
        }
        g.lineTo(-r * 0.5, r * 0.86);
        g.closePath();
        g.endFill();
        // Внутренняя огранка.
        g.lineStyle(2, light, 0.85);
        g.beginFill(dark, 0.3);
        hexPath(g, r * 0.62, Math.PI / 6);
        g.endFill();
        // Радиальные спицы от центра к углам.
        g.lineStyle(1.5, light, 0.7);
        for (let i = 0; i < 6; i++) {
            const a = i * Math.PI / 3;
            g.moveTo(0, 0);
            g.lineTo(Math.cos(a) * r, Math.sin(a) * r);
        }
        // Пылающее ядро.
        g.lineStyle(2, 0xffffff, 0.9);
        g.beginFill(light, 0.6);
        hexPath(g, r * 0.35, 0);
        g.endFill();
        g.beginFill(0xffffff, 0.95);
        g.drawCircle(0, 0, r * 0.09);
        g.endFill();
    } else if (shape === 'line') {
        const r = size * 0.6;
        const len = size * 1.8;
        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(m.color, 1);
        g.drawRoundedRect(-len / 2, -r, len, r * 2, r * 0.6);
        g.endFill();
        g.beginFill(dark, 0.25);
        g.drawRoundedRect(-len / 2, 0, len, r, r * 0.3);
        g.endFill();
        g.lineStyle(2, light, 0.7);
        g.moveTo(-len * 0.3, -r * 0.2);
        g.lineTo(len * 0.3, -r * 0.2);
        g.beginFill(light, 0.9);
        g.drawCircle(0, -r * 0.3, r * 0.15);
        g.endFill();
    } else if (shape === 'pebble') {
        // Неправильной формы камень с трещинами — отличается от круглого dot.
        const R = size * 0.95;
        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(m.color, 1);
        g.moveTo(-R * 0.9, R * 0.3);
        g.quadraticCurveTo(-R * 1.15, -R * 0.2, -R * 0.4, -R * 0.8);
        g.quadraticCurveTo(R * 0.2, -R * 1.05, R * 0.75, -R * 0.45);
        g.quadraticCurveTo(R * 1.1, R * 0.15, R * 0.6, R * 0.6);
        g.quadraticCurveTo(R * 0.05, R * 1.05, -R * 0.5, R * 0.75);
        g.closePath();
        g.endFill();
        g.lineStyle(2, dark, 0.65);
        g.moveTo(-R * 0.35, 0);
        g.lineTo(-R * 0.05, R * 0.4);
        g.lineTo(R * 0.45, R * 0.28);
        g.moveTo(0, -R * 0.25);
        g.lineTo(R * 0.3, -R * 0.6);
        g.lineStyle(0);
        g.beginFill(dark, 0.28);
        g.drawEllipse(0, R * 0.35, R * 0.55, R * 0.18);
        g.endFill();
    } else if (shape === 'dart') {
        // Стройный острый дротик с наконечником и хвостовым оперением.
        const ln = size * 2.2;
        const w = size * 0.4;
        const h = size * 0.7;
        g.lineStyle(2, 0x000000, 0.5);
        g.beginFill(m.color, 1);
        g.drawRoundedRect(-ln / 2, -w / 2, ln, w, w / 2);
        g.endFill();
        g.beginFill(dark, 0.3);
        g.drawRoundedRect(-ln / 2, 0, ln, w / 2, w / 2);
        g.endFill();
        g.beginFill(light, 1);
        g.moveTo(ln / 2, 0);
        g.lineTo(ln / 2 - h, -w);
        g.lineTo(ln / 2 - h, w);
        g.closePath();
        g.endFill();
        g.beginFill(light, 0.55);
        g.moveTo(-ln / 2, 0);
        g.lineTo(-ln / 2 - h * 0.4, -w * 1.6);
        g.lineTo(-ln / 2 + h * 0.3, 0);
        g.lineTo(-ln / 2 - h * 0.4, w * 1.6);
        g.closePath();
        g.endFill();
    } else if (shape === 'spinner') {
        // Волчок-гироскоп: кольцо со спицами и центральной осью.
        const R = size * 1.3;
        const r2 = R * 0.45;
        g.lineStyle(3, 0x000000, 0.5);
        g.lineStyle(R * 0.14, m.color, 1);
        g.drawCircle(0, 0, R);
        g.endFill();
        g.lineStyle(2, light, 0.75);
        g.drawCircle(0, 0, R * 0.72);
        g.lineStyle(4, m.color, 0.95);
        g.moveTo(-r2, 0); g.lineTo(r2, 0);
        g.moveTo(0, -r2); g.lineTo(0, r2);
        g.lineStyle(0);
        g.beginFill(light, 1);
        g.drawCircle(0, 0, R * 0.12);
        g.endFill();
    } else if (shape === 'void') {
        // Чёрная дыра: аккреционный диск, горизонт событий, выбросы и осколки.
        const R = size * 1.35;
        // Глубокий тёмный ореол-бездна.
        g.lineStyle(0);
        g.beginFill(0x0a0018, 0.95);
        g.drawCircle(0, 0, R * 1.35);
        g.endFill();
        // Внешний светящийся вихрь (аккреционный диск).
        g.lineStyle(R * 0.16, col, 0.9);
        g.drawCircle(0, 0, R);
        g.lineStyle(R * 0.06, light, 0.5);
        g.drawCircle(0, 0, R);
        // Спиральные рукава диска (три дуги).
        g.lineStyle(R * 0.09, col, 0.8);
        for (let k = 0; k < 3; k++) {
            const a0 = k * (Math.PI * 2 / 3);
            g.arc(0, 0, R * 0.85, a0, a0 + 1.6);
        }
        // Фотонное кольцо вокруг горизонта событий.
        g.lineStyle(R * 0.05, 0xffffff, 0.85);
        g.drawCircle(0, 0, R * 0.42);
        // Чёрная дыра.
        g.lineStyle(0);
        g.beginFill(0x000000, 1);
        g.drawCircle(0, 0, R * 0.32);
        g.endFill();
        // Трепещущие «выбросы» на кромке горизонта.
        g.lineStyle(2, light, 0.75);
        for (let i = 0; i < 10; i++) {
            const a = i * 0.63;
            const w = 0.32 + (i % 3) * 0.05;
            g.moveTo(Math.cos(a) * R * w, Math.sin(a) * R * w);
            g.lineTo(Math.cos(a + 0.25) * R * (w + 0.05), Math.sin(a + 0.25) * R * (w + 0.05));
        }
        // Орбитальные частицы-осколки.
        g.lineStyle(0);
        g.beginFill(light, 0.9);
        for (let i = 0; i < 4; i++) {
            const a = i * 1.7;
            g.drawCircle(Math.cos(a) * R * 0.68, Math.sin(a) * R * 0.68, R * 0.06);
        }
        g.endFill();
        g.beginFill(col, 0.85);
        for (let i = 0; i < 4; i++) {
            const a = i * 1.7 + 0.6;
            g.drawCircle(Math.cos(a) * R * 1.06, Math.sin(a) * R * 1.06, R * 0.05);
        }
        g.endFill();
        } else if (shape === 'blackhole') {
            // Чёрная дыра: аккреционный диск, фотонное кольцо и горизонт событий.
            const R = size * 1.6;
            const t = performance.now() / 1000;
            g.lineStyle(0);
            // Аккреционный диск (искажённый).
            g.lineStyle(1.5, light, 0.3);
            g.drawEllipse(0, 0, R * 1.4, R * 0.7);
            g.lineStyle(2.5, light, 0.7);
            g.drawEllipse(0, 0, R * 0.9, R * 0.45);
            // Фотонное кольцо.
            g.lineStyle(1.2, 0xffffff, 0.9);
            g.drawCircle(0, 0, R * 0.42);
            // Горизонт событий.
            g.beginFill(0x000000, 1);
            g.drawCircle(0, 0, R * 0.32);
            g.endFill();
            // Вращающиеся частицы.
            g.beginFill(light, 0.9);
            for (let i = 0; i < 8; i++) {
                const a = t * 0.8 + i * Math.PI / 4;
                g.drawCircle(Math.cos(a) * R * 0.9, Math.sin(a) * R * 0.45, R * 0.06);
            }
            g.endFill();
            g.beginFill(col, 0.8);
            for (let i = 0; i < 6; i++) {
                const a = t * 0.6 + i * Math.PI / 3;
                g.drawCircle(Math.cos(a) * R * 1.1, Math.sin(a) * R * 0.55, R * 0.05);
            }
            g.endFill();
        } else if (shape === 'entropy') {
        // Хаос: рваная многолучевая звезда, трещины и бурлящее ядро.
        const R = size * 1.4;
        g.lineStyle(0);
        g.beginFill(col, 0.25);
        g.drawCircle(0, 0, R * 1.5);
        g.endFill();
        // Рваный контур из лучей разной длины.
        const spikes = 19;
        g.lineStyle(2.5, 0x000000, 0.6);
        g.beginFill(m.color, 1);
        g.moveTo(R, 0);
        for (let i = 1; i <= spikes; i++) {
            const a = (i / spikes) * Math.PI * 2;
            const rr = R * (0.45 + ((i * 13) % 7) * 0.13);
            g.lineTo(Math.cos(a) * rr, Math.sin(a) * rr);
        }
        g.closePath();
        g.endFill();
        // Хаотичные изломанные трещины-молнии.
        g.lineStyle(1.5, light, 0.75);
        for (let i = 0; i < 9; i++) {
            const a = i * 0.7 + 0.3;
            const r1 = R * (0.15 + ((i * 5) % 3) * 0.1);
            const r2 = R * (0.55 + ((i * 3) % 5) * 0.12);
            const r3 = R * (0.35 + ((i * 7) % 4) * 0.1);
            g.moveTo(Math.cos(a) * r1, Math.sin(a) * r1);
            g.lineTo(Math.cos(a + 0.5) * r2, Math.sin(a + 0.5) * r2);
            g.lineTo(Math.cos(a + 1.0) * r3, Math.sin(a + 1.0) * r3);
        }
        // Бурлящее ядро-глаз.
        g.lineStyle(0);
        g.beginFill(0x000000, 0.92);
        g.drawCircle(0, 0, R * 0.32);
        g.endFill();
        g.lineStyle(2, light, 0.9);
        g.drawCircle(0, 0, R * 0.32);
        g.lineStyle(0);
        g.beginFill(light, 0.95);
        g.drawCircle(0, 0, R * 0.15);
        g.endFill();
        // Осколки-спутники, летающие вокруг.
        g.beginFill(light, 0.85);
        for (let i = 0; i < 7; i++) {
            const a = i * 0.9;
            g.drawCircle(Math.cos(a) * R * 1.14, Math.sin(a) * R * 1.14, R * (0.045 + (i % 3) * 0.04));
        }
        g.endFill();
    } else if (shape === 'arrow') {
        // Стрела: длинное древко + наконечник.
        const ln = size * 2.0;
        const w = size * 0.55;
        const h = size * 0.95;
        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(m.color, 1);
        g.drawRoundedRect(-ln / 2, -w / 2, ln, w, w / 3);
        g.endFill();
        g.beginFill(dark, 0.25);
        g.drawRoundedRect(-ln / 2, 0, ln, w / 2, w / 3);
        g.endFill();
        g.beginFill(light, 1);
        g.moveTo(ln / 2, 0);
        g.lineTo(ln / 2 - h, -w);
        g.lineTo(ln / 2 - h, w);
        g.closePath();
        g.endFill();
        g.beginFill(m.color, 0.9);
        g.moveTo(-ln / 2, 0);
        g.lineTo(-ln / 2 + h * 0.6, -w);
        g.lineTo(-ln / 2 + h * 0.6, w);
        g.closePath();
        g.endFill();
    } else if (shape === 'block') {
        // Тяжёлый каменный блок с металлическими скобами.
        const s = size * 1.0;
        g.lineStyle(4, 0x000000, 0.6);
        g.beginFill(m.color, 1);
        g.drawRoundedRect(-s, -s, s * 2, s * 2, 6);
        g.endFill();
        g.lineStyle(3, light, 0.8);
        const b = s * 0.72;
        for (const [sx, sy] of [[-1, -1], [1, -1], [-1, 1], [1, 1]]) {
            g.moveTo(sx * b, sy * b);
            g.lineTo(sx * b, sy * (b - s * 0.2));
            g.lineTo(sx * (b - s * 0.2), sy * b);
        }
        g.lineStyle(0);
        g.beginFill(0xffffff, 0.45);
        g.drawCircle(0, 0, s * 0.2);
        g.endFill();
    } else if (shape === 'shield') {
        // Щит: геральдическая форма с окантовкой и символом.
        const r = size * 1.05;
        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(m.color, 1);
        g.moveTo(0, -r);
        g.quadraticCurveTo(r * 0.8, -r * 0.5, r * 0.8, r * 0.2);
        g.lineTo(0, r * 1.0);
        g.lineTo(-r * 0.8, r * 0.2);
        g.quadraticCurveTo(-r * 0.8, -r * 0.5, 0, -r);
        g.closePath();
        g.endFill();
        g.lineStyle(2, light, 0.9);
        g.moveTo(0, -r * 0.7);
        g.quadraticCurveTo(r * 0.55, -r * 0.4, r * 0.55, r * 0.15);
        g.lineTo(0, r * 0.7);
        g.lineTo(-r * 0.55, r * 0.15);
        g.quadraticCurveTo(-r * 0.55, -r * 0.4, 0, -r * 0.7);
        g.closePath();
        g.lineStyle(0);
        g.beginFill(light, 0.9);
        g.drawCircle(0, 0, r * 0.16);
        g.endFill();
    } else if (shape === 'hive') {
        // Улей: шестиугольник с сотами и маткой в центре.
        const r = size * 1.1;
        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(m.color, 1);
        g.moveTo(r, 0);
        for (let i = 1; i <= 6; i++) {
            const a = i * Math.PI / 3;
            g.lineTo(r * Math.cos(a), r * Math.sin(a));
        }
        g.endFill();
        g.lineStyle(1.5, dark, 0.75);
        for (const [ox, oy, s] of [[0, -r * 0.5, r * 0.42], [-r * 0.37, 0, r * 0.42], [r * 0.37, 0, r * 0.42], [0, r * 0.5, r * 0.42]]) {
            g.moveTo(ox + s, oy);
            for (let i = 1; i <= 6; i++) {
                const a = i * Math.PI / 3;
                g.lineTo(ox + s * Math.cos(a), oy + s * Math.sin(a));
            }
        }
        g.lineStyle(0);
        g.beginFill(light, 0.9);
        g.drawCircle(0, -r * 0.5, r * 0.13);
        g.endFill();
    } else if (shape === 'spark') {
        // Вспышка фрактальной искры: четырёхлучевая звезда с ядром.
        const r = size * 1.15;
        g.lineStyle(3, 0x000000, 0.6);
        g.beginFill(m.color, 1);
        g.moveTo(0, -r);
        g.quadraticCurveTo(r * 0.35, -r * 0.35, r, 0);
        g.quadraticCurveTo(r * 0.35, r * 0.35, 0, r);
        g.quadraticCurveTo(-r * 0.35, r * 0.35, -r, 0);
        g.quadraticCurveTo(-r * 0.35, -r * 0.35, 0, -r);
        g.closePath();
        g.endFill();
        g.lineStyle(0);
        g.beginFill(light, 0.9);
        g.drawCircle(0, 0, r * 0.3);
        g.endFill();
        g.lineStyle(2, light, 0.7);
        for (let i = 0; i < 4; i++) {
            const a = (i * Math.PI) / 2 + Math.PI / 4;
            g.moveTo(Math.cos(a) * r * 0.9, Math.sin(a) * r * 0.9);
            g.lineTo(Math.cos(a) * r * 1.35, Math.sin(a) * r * 1.35);
        }
    } else if (shape === 'wall') {
        // Фрактальная стена: массивная плита с кирпичными швами.
        const w = size * 1.4;
        const h = size * 0.9;
        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(m.color, 1);
        g.drawRoundedRect(-w, -h, w * 2, h * 2, 4);
        g.endFill();
        g.lineStyle(1.5, dark, 0.65);
        g.moveTo(-w, -h * 0.5); g.lineTo(w, -h * 0.5);
        g.moveTo(-w, h * 0.5); g.lineTo(w, h * 0.5);
        g.moveTo(-w * 0.35, -h * 0.5); g.lineTo(-w * 0.35, h * 0.5);
        g.moveTo(w * 0.35, -h); g.lineTo(w * 0.35, -h * 0.5);
        g.moveTo(-w * 0.35, h * 0.5); g.lineTo(-w * 0.35, h);
        g.moveTo(w * 0.35, h * 0.5); g.lineTo(w * 0.35, h);
        g.lineStyle(0);
        g.beginFill(light, 0.8);
        for (const [sx, sy] of [[-1, -1], [1, -1], [-1, 1], [1, 1]]) {
            g.drawCircle(sx * (w - 5), sy * (h - 5), 3);
        }
        g.endFill();
    } else if (shape === 'tank') {
        // ФРАГТАНК (главный моб Fractal-зоны): фиолетовый танк — гусеницы
        // по бокам, скошенный корпус с энергетическими швами, башня-купол
        // с пушкой (поворачивается «сканированием») и рунным ядром.
        const s = size;
        const hw = s * 1.55; // полуширина корпуса (вдоль движения)
        const hh = s * 1.05; // полувысота корпуса
        const tw = s * 0.5;  // ширина гусеницы
        const t = performance.now() * 0.003;
        const pulse = 0.5 + 0.5 * Math.sin(t);
        // Гусеницы: тёмные ленты с поперечными траками.
        g.lineStyle(3, 0x000000, 0.6);
        g.beginFill(0x161220, 1);
        g.drawRoundedRect(-hw - s * 0.1, -hh - tw * 0.9, (hw + s * 0.1) * 2, tw, tw * 0.45);
        g.drawRoundedRect(-hw - s * 0.1, hh + tw * 0.9 - tw, (hw + s * 0.1) * 2, tw, tw * 0.45);
        g.endFill();
        g.lineStyle(0);
        g.beginFill(0x3d3452, 1);
        for (let i = 0; i < 7; i++) {
            const tx = -hw + (i * (hw * 2 - s * 0.16)) / 6;
            g.drawRoundedRect(tx, -hh - tw * 0.9, s * 0.12, tw, 2);
            g.drawRoundedRect(tx, hh + tw * 0.9 - tw, s * 0.12, tw, 2);
        }
        // Корпус: скошенная броня (нос и корма срезаны).
        g.lineStyle(3, 0x000000, 0.65);
        g.beginFill(m.color, 1);
        g.moveTo(-hw, -hh * 0.55);
        g.lineTo(-hw * 0.55, -hh);
        g.lineTo(hw * 0.55, -hh);
        g.lineTo(hw, -hh * 0.55);
        g.lineTo(hw, hh * 0.55);
        g.lineTo(hw * 0.55, hh);
        g.lineTo(-hw * 0.55, hh);
        g.lineTo(-hw, hh * 0.55);
        g.closePath();
        g.endFill();
        // Броневые фаски: светлая верхняя половина, тёмная нижняя.
        g.lineStyle(0);
        g.beginFill(light, 0.24);
        g.moveTo(-hw, -hh * 0.55);
        g.lineTo(-hw * 0.55, -hh);
        g.lineTo(hw * 0.55, -hh);
        g.lineTo(hw, -hh * 0.55);
        g.lineTo(hw, 0);
        g.lineTo(-hw, 0);
        g.closePath();
        g.endFill();
        g.beginFill(dark, 0.4);
        g.moveTo(-hw, 0);
        g.lineTo(hw, 0);
        g.lineTo(hw, hh * 0.55);
        g.lineTo(hw * 0.55, hh);
        g.lineTo(-hw * 0.55, hh);
        g.lineTo(-hw, hh * 0.55);
        g.closePath();
        g.endFill();
        // Энергетические швы на корпусе (пульсируют) — «фрактальная» начинка.
        g.lineStyle(2.2, light, 0.45 + 0.45 * pulse);
        g.moveTo(-hw * 0.75, -hh * 0.3);
        g.lineTo(-hw * 0.2, -hh * 0.3);
        g.moveTo(-hw * 0.75, hh * 0.3);
        g.lineTo(-hw * 0.2, hh * 0.3);
        g.moveTo(hw * 0.2, -hh * 0.3);
        g.lineTo(hw * 0.75, -hh * 0.3);
        g.moveTo(hw * 0.2, hh * 0.3);
        g.lineTo(hw * 0.75, hh * 0.3);
        // Пушка: башня наведена СЕРВЕРОМ на цель агрессии (turret — мировой
        // угол); в пассиве — по направлению движения. Локальный угол =
        // мировой минус текущий поворот корпуса (он сглажен клиентом).
        const bl = s * 1.55; // длина ствола
        const turretL = (m.turret || 0) - g.rotation;
        const bc = Math.cos(turretL), bs = Math.sin(turretL);
        g.lineStyle(s * 0.34, 0x1c2230, 1);
        g.moveTo(0, 0);
        g.lineTo(bc * bl, bs * bl);
        g.lineStyle(s * 0.16, 0x4a5570, 1);
        g.moveTo(0, 0);
        g.lineTo(bc * bl, bs * bl);
        g.lineStyle(0);
        g.beginFill(0x2a3040, 1);
        g.drawCircle(bc * bl, bs * bl, s * 0.2);
        g.endFill();
        g.beginFill(light, 0.5 + 0.3 * pulse);
        g.drawCircle(bc * bl, bs * bl, s * 0.09);
        g.endFill();
        // Башня-купол: тёмный обод, броня цвета моба, блик.
        g.lineStyle(3, 0x1c2230, 1);
        g.beginFill(m.color, 1);
        g.drawCircle(0, 0, s * 0.68);
        g.endFill();
        g.lineStyle(0);
        g.beginFill(light, 0.3);
        g.drawCircle(-s * 0.16, -s * 0.16, s * 0.42);
        g.endFill();
        // Руна-ядро на башне (пульс).
        g.lineStyle(1.8, light, 0.85);
        g.drawCircle(0, 0, s * 0.3);
        g.lineStyle(0);
        g.beginFill(light, 0.6 + 0.4 * pulse);
        g.drawCircle(0, 0, s * 0.14);
        g.endFill();
        g.beginFill(0xffffff, 0.9);
        g.drawCircle(0, 0, s * 0.06);
        g.endFill();
    }
    else if (shape === 'zigzag') {
        const s = size;
        const tt = performance.now() / 1000;
        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(col, 1);
        g.moveTo(s * 1.25, 0);
        g.lineTo(s * 0.2, -s * 0.55);
        g.lineTo(-s * 0.35, -s * 0.28);
        g.lineTo(-s * 1.05, -s * 0.62);
        g.lineTo(-s * 0.55, 0);
        g.lineTo(-s * 1.05, s * 0.62);
        g.lineTo(-s * 0.35, s * 0.28);
        g.lineTo(s * 0.2, s * 0.55);
        g.closePath();
        g.endFill();
        g.lineStyle(0);
        g.beginFill(dark, 0.35);
        g.moveTo(s * 1.25, 0);
        g.lineTo(s * 0.2, -s * 0.55);
        g.lineTo(-s * 0.55, 0);
        g.closePath();
        g.endFill();
        // Пульсирующая искра в сердце.
        const pul = 0.6 + 0.4 * Math.sin(tt * 9);
        g.beginFill(0xfff3c0, pul);
        g.drawCircle(s * 0.25, 0, s * 0.24 * pul + s * 0.08);
        g.endFill();
        g.lineStyle(2, light, 0.9);
        g.moveTo(-s * 0.9, -s * 0.4); g.lineTo(-s * 0.45, 0); g.lineTo(-s * 0.9, s * 0.4);
        g.lineStyle(0);
        return;
    }
    else if (shape === 'quarry') {
        const s = size * 0.95;
        const tt = performance.now() / 1000;
        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(col, 1);
        g.drawRoundedRect(-s, -s * 0.9, s * 2, s * 1.8, s * 0.14);
        g.endFill();
        g.lineStyle(0);
        g.beginFill(0x000000, 0.22);
        g.drawRoundedRect(-s, -s * 0.9, s * 2, s * 0.85, s * 0.14);
        g.endFill();
        // Кладка: швы.
        g.lineStyle(2, 0x000000, 0.4);
        for (let i = -1; i <= 1; i++) {
            g.moveTo(-s * 0.8, i * s * 0.42);
            g.lineTo(s * 0.8, i * s * 0.42);
        }
        for (let i = -1; i <= 1; i++) {
            const off = (i % 2 === 0) ? s * 0.42 : 0;
            g.moveTo(off, -s * 0.9); g.lineTo(off, -s * 0.48);
            g.moveTo(off - s * 0.42, s * 0.48); g.lineTo(off - s * 0.42, s * 0.9);
        }
        // Янтарная руда: три вкрапления, мерцают вразнобой.
        for (let i = 0; i < 3; i++) {
            const gx = (i - 1) * s * 0.55;
            const gy = (i === 1) ? -s * 0.15 : s * 0.35;
            const gl = 0.55 + 0.45 * Math.sin(tt * 2.2 + i * 2.1);
            g.beginFill(0xffc95e, gl);
            g.drawCircle(gx, gy, s * 0.13);
            g.endFill();
            g.beginFill(0xffffff, gl * 0.5);
            g.drawCircle(gx - s * 0.04, gy - s * 0.04, s * 0.05);
            g.endFill();
        }
        // Трещина.
        g.lineStyle(2, 0x1a1210, 0.6);
        g.moveTo(-s * 0.3, -s * 0.9);
        g.lineTo(-s * 0.1, -s * 0.35); g.lineTo(-s * 0.42, s * 0.1); g.lineTo(-s * 0.15, s * 0.5);
        g.lineStyle(0);
        return;
    }
    else if (shape === 'nexus') {
        // Nexus («ИИ-игрок»): тело БУКВАЛЬНО как у игрока — та же отрисовка
        // шара, что в drawPlayerSprite (свечение, круг, блик, тёмный обод,
        // светлая окантовка). Цвет — из m.color (скин из палитры меню).
        const R = size;
        g.beginFill(m.color, 0.22);
        g.drawCircle(0, 0, R + 6);
        g.endFill();
        g.beginFill(m.color, 1);
        g.drawCircle(0, 0, R);
        g.endFill();
        g.beginFill(light, 0.55);
        g.drawEllipse(-R * 0.28, -R * 0.38, R * 0.52, R * 0.32);
        g.endFill();
        g.lineStyle(2, dark, 0.9);
        g.drawCircle(0, 0, R);
        g.lineStyle(1.5, petalShade(m.color, 1.7), 0.95);
        g.drawCircle(0, 0, R + 1);
        g.lineStyle(0);
    } else {
        // Circle
        const R = size;
        g.lineStyle(size * 0.10, light, 0.15);
        g.drawCircle(0, 0, R * 1.05);
        g.lineStyle(0);
        g.lineStyle(size * 0.12, 0x000000, 0.9);
        g.beginFill(m.color, 1);
        g.drawCircle(0, 0, R);
        g.endFill();
        g.beginFill(dark, 0.55);
        g.drawCircle(0, R * 0.16, R * 0.90);
        g.endFill();
        g.lineStyle(0);
        g.beginFill(0xffffff, 0.7);
        g.drawCircle(-R * 0.25, -R * 0.33, R * 0.42);
        g.endFill();
        g.beginFill(m.color, 0.6);
        g.drawCircle(-R * 0.30, -R * 0.27, R * 0.38);
        g.endFill();
        g.lineStyle(size * 0.07, light, 0.5);
        g.drawCircle(0, R * 0.08, R * 0.55);
        g.lineStyle(0);
        g.beginFill(0x000000, 0.10);
        g.drawEllipse(0, R * 0.34, R * 0.52, R * 0.14);
        g.endFill();
    }

    // Нос-указатель направления движения: вся фигура поворачивается к dirX/dirY,
    // а этот острый шип показывает куда именно моб идёт сейчас. Неподвижные
    // мобы (hive/block, speed=0) нос не рисуем; у треугольника своя
    // стрелка-очерк из угла, общий нос не нужен.
    if (!m.static && m.speed > 0 && shape !== 'triangle') {
        g.lineStyle(0);
        g.beginFill(light, 1);
        g.moveTo(size * 1.35, 0);
        g.lineTo(size * 0.95, -size * 0.30);
        g.lineTo(size * 1.08, 0);
        g.lineTo(size * 0.95, size * 0.30);
        g.closePath();
        g.endFill();
    }
}

// Создаёт «затухающий труп» моба: когда моб убит, вместо мгновенного исчезания
// показываем постепенное растворение (альфа/размер) на месте смерти.
function spawnMobFade(msg) {
    if (!msg || !msg.id) return;
    const live = mobActors[msg.id];
    const c = new PIXI.Container();
    const g = new PIXI.Graphics();
    drawMobShape(g, { shape: msg.shape || 'circle', size: msg.size || 20, color: msg.color != null ? msg.color : 0x888888 });
    c.addChild(g);
    c.position.set(msg.x, msg.y);
    // Сохраняем угол моба каким он был за мгновение до смерти (последний живой кадр).
    c.rotation = live ? live.g.rotation : 0;
    c.alpha = 1;
    world.addChild(c);
    mobFades.set(msg.id, { c, t: 0 });
    // Если рядом был "живой" актор того же моба — убираем его (моб мёртв).
    if (live) {
        world.removeChild(live.c);
        delete mobActors[msg.id];
    }
}

// shade смешивает цвет hex с белым (+amt) или чёрным (-amt). Возвращает hex 0xRRGGBB.
function shade(hex, amt) {    let r = (hex >> 16) & 255;
    let g = (hex >> 8) & 255;
    let b = hex & 255;
    if (amt >= 0) {
        r += (255 - r) * (amt / 100);
        g += (255 - g) * (amt / 100);
        b += (255 - b) * (amt / 100);
    } else {
        r += r * (amt / 100);
        g += g * (amt / 100);
        b += b * (amt / 100);
    }
    return (Math.round(r) << 16) | (Math.round(g) << 8) | Math.round(b);
}

function redrawMobHP(a, m) {
    const bw = a.bw, bh = a.bh;
    const pct = Math.max(0, m.hp / m.maxHp);
    const fillW = Math.max(1, bw * pct);

    // Эхо: при уроне светлый след остаётся на старой ширине, потом тает.
    if (fillW < a.fillW - 0.5) a.ghost = Math.max(a.ghost, a.fillW);
    else if (fillW > a.fillW + 0.5) a.ghost = fillW;
    a.fillW = fillW;

    const aColor = m.ally;
    const col = aColor ? 0x3ddc84 : (pct > 0.6 ? 0x3ddc84 : pct > 0.3 ? 0xf1c40f : 0xe74c3c);
    const rad = Math.min(bh / 2, fillW / 2);

    // Двухслойный заполнитель: тёмная обводка + более светлое ядро → объём.
    const colOut = shade(col, -14);
    const colCore = shade(col, 22);
    a.barFill.clear();
    a.barFill.beginFill(colOut, 1);
    a.barFill.drawRoundedRect(0, 0, fillW, bh, rad);
    a.barFill.endFill();
    a.barFill.beginFill(colCore, 1);
    a.barFill.drawRoundedRect(0, 0.9, Math.max(1, fillW), Math.max(0.5, bh - 1.8),
        Math.min(rad, 1.4));
    a.barFill.endFill();

    // Глянец-шапка сверху заполнителя.
    a.barShine.clear();
    a.barShine.beginFill(0xffffff, 0.3);
    a.barShine.drawRoundedRect(0, 0, fillW, bh * 0.42, Math.min(bh * 0.42 / 2, fillW / 2));
    a.barShine.endFill();

    // Окантовка всего бара — в цвет редкости моба (у союзника — зелёная).
    // Подвид моба (gold/ring/star/crescent): обводка бара в цвет подвида,
    // у золотого — потолще.
    const VAR_COLORS = { gold: 0xffd700, ring: 0xf1c40f, star: 0xfffacd, crescent: 0xb0c4de };
    const isGold = a.variant === 'gold';
    const edgeCol = a.variant ? (VAR_COLORS[a.variant] || 0xffd700) : (aColor ? 0x3ddc84 : rarityNum(a.rarityI));
    a.barEdge.clear();
    a.barEdge.lineStyle(isGold ? 3 : 1, edgeCol, 0.9);
    a.barEdge.drawRoundedRect(isGold ? -2 : 0.5, isGold ? -2 : 0.5, bw + (isGold ? 4 : 0), bh + (isGold ? 4 : 0), (bh + (isGold ? 4 : 0)) / 2);
    a.barEdge.lineStyle(0);
}

// ----- Отрисовка орбитальных петалей (кружатся вокруг игрока) -----
function renderPetals(petals) {
    const seen = new Set();
    saturnAlive = {};
    let myLineRar = -1; // редкость живой line у моего игрока (для множителя обзора)
    for (const pt of petals) {
        seen.add(pt.id);
        // Saturn — НЕ «петаль»: рисуется как 3D-кольцо вокруг игрока (в render).
        // Здесь только запоминаем живость (можно ли показать кольцо) и пропускаем.
        if (pt.kind === 'saturn') {
            const recharging = pt.cd != null && pt.cd > 0 && pt.cdMax != null && pt.cdMax >= 1.0;
            if (pt.hp > 0 && !recharging) {
                const idx = pt.id.indexOf('_h');
                if (idx > 4) {
                    const owner = pt.id.slice(4, idx);
                    saturnAlive[owner] = Math.max(saturnAlive[owner] || 0, pt.rarity || 0);
                }
            }
            continue;
        }
        let a = petalActors[pt.id];
        if (!a) {
            const g = new PIXI.Graphics();
            world.addChild(g);
            a = { g, targetX: pt.x, targetY: pt.y, lastHP: pt.hp, hit: 0, size: pt.size || 8 };
            a.hitRing = new PIXI.Graphics();
            g.addChild(a.hitRing);
            // Оверлей хитбокса петали игрока (включается в настройках).
            a.hitG = new PIXI.Graphics();
            g.addChildAt(a.hitG, 0);
            petalActors[pt.id] = a;
        }
        // Хитбокс петали игрока: круг радиусом pt.size.
        if (a.hitG) {
            a.hitG.clear();
            if (showHitPlayerPetals) {
                a.hitG.lineStyle(1.5, 0x66ddff, 0.75);
                a.hitG.drawCircle(0, 0, pt.size);
                a.hitG.lineStyle(0);
            }
        }
        a.targetX = pt.x;
        a.targetY = pt.y;
        // Уникальные аксессуары (eye, line) у МОЕГО игрока висят прямо в центре —
        // это «прицепленные» к игроку приборы, не орбиталы.
        const isMine = !!(myId && pt.id && pt.id.indexOf('pet_' + myId + '_h') === 0);
        const isCenterPetal = pt.kind === 'eye' || pt.kind === 'line' || pt.kind === 'spike';
        if (isCenterPetal && isMine) {
            a.targetX = myPlayerX;
            a.targetY = myPlayerY;
        }
        drawPetalShape(a.g, pt);
        // Eye: фиксированный визуальный размер на игроке — растёт только
        // эффект (радиус орбиты) и цвет, не сам спрайт.
        if (pt.kind === 'eye') a.g.scale.set(16 / Math.max(10, pt.size));
        else a.g.scale.set(1);
        // Клинок (blade): остриём к игроку (рукоять наружу) — «наоборот» от радиуса.
        // Компас (compass): кружок со стрелкой повёрнут по мировому углу на цель.
        // Шип (spike, аксессуар на игроке): непрерывное вращение шипов.
        let gRot = 0;
        if (pt.kind === 'blade') gRot = (pt.angle || 0) + Math.PI;
        else if (pt.kind === 'twirl') gRot = (pt.angle || 0) + Math.PI / 2;
        else if (pt.kind === 'compass' && isMine && compassInfo && compassInfo.on) gRot = compassInfo.a || 0;
        else if (pt.kind === 'spike') gRot = (performance.now() * 0.0018) % (Math.PI * 2);
        a.g.rotation = gRot;
        // Щит (shield): по мере заряда (shieldCharge 0..1) петаль подлетает к игроку
        // и тает, пополняя полоску щита. На кулдауне (recharge) петаль скрыта.
        if (pt.kind === 'shield') {
            const recharging = pt.cd != null && pt.cd > 0 && pt.cdMax != null && pt.cdMax >= 1.0;
            if (!recharging && pt.shieldCharge != null) {
                const sc = Math.max(0, Math.min(1, pt.shieldCharge));
                a.targetX = pt.x + (myPlayerX - pt.x) * sc;
                a.targetY = pt.y + (myPlayerY - pt.y) * sc;
                a.shieldFade = 1 - sc; // 1 = на орбите, 0 = впитался в игрока
            } else {
                a.shieldFade = null;
            }
        } else if (a.shieldFade != null) {
            a.shieldFade = null;
        }
        // Стрелка компаса: рисуется поверх кружка и указывает по мировому углу
        // на ближайшего Fractal/Chaos-моба (весь a.g повёрнут на compassInfo.a).
        if (pt.kind === 'compass') {
            const ns = pt.size;
            a.g.lineStyle(1.6, 0xff3b3b, 0.9);
            a.g.moveTo(-ns * 0.34, 0);
            a.g.lineTo(ns * 0.5, 0);
            a.g.lineStyle(0);
            a.g.beginFill(0xff3b3b, 0.9);
            a.g.drawCircle(ns * 0.58, 0, ns * 0.13);
            a.g.endFill();
            a.g.beginFill(0xffffff, 0.5);
            a.g.drawCircle(-ns * 0.4, 0, ns * 0.06);
            a.g.endFill();
        }
        // Антенна-обзор (line): учитываем максимальную редкость ЖИВОЙ line у меня.
        if (pt.kind === 'line' && isMine && pt.hp > 0) {
            const rechargingL = pt.cd != null && pt.cd > 0 && pt.cdMax != null && pt.cdMax >= 1.0;
            if (!rechargingL) {
                myLineRar = Math.max(myLineRar, pt.rarity || 0);
            }
        }
        // Отравленная петаль (удар «гадюкой» в ПВП) — зелёное кольцо поверх.
        if (pt.poison) {
            a.g.lineStyle(1.5, 0x39ff5e, 0.9);
            a.g.drawCircle(0, 0, pt.size + 2);
        }
        // Кулдаун активной линейки: петель ID = "pet_<ник>_h<слот>". Запоминаем
        // долю оставшейся перезарядки, чтобы рисовать «вуаль» на хотбаре.
        if (myId) {
            const pref = 'pet_' + myId + '_h';
            if (pt.id && pt.id.indexOf(pref) === 0) {
                const slot = parseInt(pt.id.slice(pref.length), 10);
                const max = (pt.cdMax != null && pt.cdMax > 0) ? pt.cdMax : 0;
                hotbarSlotCD[slot] = (max > 0 && pt.cd > 0) ? Math.max(0, Math.min(1, pt.cd / max)) : 0;
            }
        }
        // Индикатор потери HP: при уроне петаль вспыхивает красным и мигает.
        const ratio = pt.hp / pt.maxHp;
        const dmgFlash = a.lastHP > pt.hp;
        a.lastHP = pt.hp;
        if (dmgFlash) a.flash = 1;
        else if (a.flash && a.flash > 0) a.flash *= 0.92;
// Петаль пропадает ТОЛЬКО когда по-настоящему сломалась (полная перезарядка,
        // cdMax >= 1 с). Исчезновение/возврат — плавные (лерп прозрачности),
        // а не мгновенные. Короткий кулдаун после удара (0.3 с) не прячет
        // петаль — вместо этого вокруг неё мигает кольцо «эффекта удара».
        const recharging = pt.cd != null && pt.cd > 0 && pt.cdMax != null && pt.cdMax >= 1.0;
        const targetAlpha = (recharging && !(isCenterPetal && isMine)) ? 0 : 1;
        a.g.alpha += (targetAlpha - a.g.alpha) * 0.22;
        if (a.g.alpha < 0.03) a.g.alpha = 0;
        if (a.g.alpha > 0.97) a.g.alpha = 1;
        if (recharging) {
            a.hit = 0;
            if (a.hitRing) a.hitRing.clear();
        } else if (pt.cd != null && pt.cd > 0) {
            a.hit = 1;
        }
        if (a.flash > 0.05) {
            const flash = new PIXI.Graphics();
            flash.beginFill(0xff4444, 0.7 * a.flash);
            flash.drawCircle(0, 0, pt.size + 3);
            flash.endFill();
            a.g.addChild(flash);
            a.flashSprite = flash;
            setTimeout(() => { if (a.flashSprite) a.flashSprite.destroy(); }, 120);
        }
    }
    for (const id of Object.keys(petalActors)) {
        if (!seen.has(id)) {
            fadeOutThenRemove(petalActors[id].g);
            delete petalActors[id];
        }
    }
    // Целевой множитель обзора line (плавная анимация — в tick(), 60 раз/с).
    myLineTarget = myLineRar >= 0 ? Math.pow(1.1, myLineRar) : 1;
}

// Плавное растворение сущностей, пропавших из state (ушли за край камеры,
// умерли, ушли из зоны): 260 мс fade-out, затем удаление из сцены.
const fadeOuts = [];
function fadeOutThenRemove(node) {
    if (!node || !node.parent) return;
    node.alpha = node.alpha || 1;
    fadeOuts.push({ node, t: 0 });
}
function tickFadeOuts(ms) {
    for (let i = fadeOuts.length - 1; i >= 0; i--) {
        const f = fadeOuts[i];
        f.t += ms;
        const k = 1 - f.t / 260;
        if (k <= 0) {
            if (f.node.parent) f.node.parent.removeChild(f.node);
            fadeOuts.splice(i, 1);
        } else {
            f.node.alpha = k;
        }
    }
}

// Затемнение/осветление цвета редкости для деталей петаля (f<1 — темнее, >1 — светлее).
function petalShade(hex, f) {    const r = (hex >> 16) & 255, g = (hex >> 8) & 255, b = hex & 255;
    const ch = v => Math.max(0, Math.min(255, Math.round(v * f)));
    return (ch(r) << 16) | (ch(g) << 8) | ch(b);
}

// Правильный многоугольник (n сторон) с радиусом rr, начиная с правой вершины.
function petalPoly(g, n, rr) {
    g.moveTo(rr, 0);
    for (let i = 1; i < n; i++) {
        const a = (i / n) * Math.PI * 2;
        g.lineTo(Math.cos(a) * rr, Math.sin(a) * rr);
    }
    g.closePath();
}

// drawPetalShape рисует вид петаля (лепесток) с цветом редкости.
// У каждого вида свой уникальный силуэт.
function drawPetalShape(g, pt) {
    g.clear();
    if (g.children.length) g.removeChildren();
    const hex = pt.color != null ? pt.color : 0xecf0f1;
    const s = pt.size;
    const kind = pt.kind || 'oval';
    const dark = petalShade(hex, 0.62);
    const light = petalShade(hex, 1.35);

    switch (kind) {
        case 'rhombus': {
            // Ромб-кристалл: гранёный лозендж с внутренней подсветкой ядра.
            g.lineStyle(1.5, 0xffffff, 0.8);
            g.beginFill(hex);
            g.moveTo(0, -s * 1.25);
            g.lineTo(s * 0.85, 0);
            g.lineTo(0, s * 1.25);
            g.lineTo(-s * 0.85, 0);
            g.closePath();
            g.endFill();
            // Фасетная полутень на левой половине.
            g.beginFill(dark, 0.3);
            g.moveTo(0, -s * 1.25);
            g.lineTo(-s * 0.85, 0);
            g.lineTo(0, s * 1.25);
            g.closePath();
            g.endFill();
            // Хребет (замедляющие линии-«мороз» по вертикали).
            g.lineStyle(1, light, 0.7);
            g.moveTo(0, -s * 1.25);
            g.lineTo(0, s * 1.25);
            g.lineStyle(0);
            // Ядро-искра.
            g.beginFill(light, 0.85);
            g.drawCircle(0, 0, s * 0.22);
            g.endFill();
            break;
        }
        case 'twirl': {
            // Вертушка: вытянутый ромб-дротик с наконечниками в обе стороны.
            // Ось вдоль орбиты (поворот см. renderPetals) — разворот направления
            // вращения орбиты читается визуально.
            g.lineStyle(1.5, 0xffffff, 0.8);
            g.beginFill(hex);
            g.moveTo(0, -s * 1.45);
            g.lineTo(s * 0.58, 0);
            g.lineTo(0, s * 1.45);
            g.lineTo(-s * 0.58, 0);
            g.closePath();
            g.endFill();
            // Затемнение верхней грани.
            g.beginFill(dark, 0.3);
            g.moveTo(0, -s * 1.45);
            g.lineTo(0, 0);
            g.lineTo(-s * 0.58, 0);
            g.closePath();
            g.endFill();
            // Хребет-наконечники по оси.
            g.lineStyle(1.2, light, 0.75);
            g.moveTo(0, -s * 1.15);
            g.lineTo(0, s * 1.15);
            g.lineStyle(0);
            // Ядро.
            g.beginFill(light, 0.85);
            g.drawCircle(0, 0, s * 0.16);
            g.endFill();
            break;
        }
        case 'oval': {
            // Лепесток.
            g.lineStyle(1.5, 0xffffff, 0.8);
            g.beginFill(hex);
            g.moveTo(0, -s * 1.1);
            g.quadraticCurveTo(s, 0, 0, s * 1.1);
            g.quadraticCurveTo(-s, 0, 0, -s * 1.1);
            g.endFill();
            // Средняя жилка.
            g.lineStyle(1, dark, 0.5);
            g.moveTo(0, -s * 0.9);
            g.quadraticCurveTo(s * 0.15, 0, -s * 0.1, s * 0.85);
            g.lineStyle(0);
            break;
        }
        case 'thorns': {
            // Колючка-треугольник с остриём вверх.
            g.lineStyle(1.5, 0xffffff, 0.8);
            g.beginFill(hex);
            g.moveTo(0, -s * 1.3);
            g.lineTo(s * 1.05, s * 0.95);
            g.lineTo(-s * 1.05, s * 0.95);
            g.closePath();
            g.endFill();
            g.lineStyle(1, light, 0.55);
            g.moveTo(-s * 0.05, -s * 0.85);
            g.lineTo(s * 0.3, s * 0.35);
            g.lineStyle(0);
            break;
        }
        case 'rock': {
            // Камень с нижней фаской и бликом сверху.
            g.lineStyle(1.5, 0xffffff, 0.8);
            g.beginFill(hex);
            g.drawRoundedRect(-s, -s, s * 2, s * 2, 3);
            g.endFill();
            g.lineStyle(0);
            g.beginFill(light, 0.3);
            g.drawRoundedRect(-s * 0.72, -s * 0.85, s * 1.2, s * 0.34, 2);
            g.endFill();
            g.beginFill(dark, 0.4);
            g.drawRoundedRect(-s * 0.72, s * 0.18, s * 1.44, s * 0.7, 2);
            g.endFill();
            break;
        }
        case 'ball': {
            // Мяч с бликом.
            g.lineStyle(1.5, 0xffffff, 0.8);
            g.beginFill(hex);
            g.drawCircle(0, 0, s);
            g.endFill();
            g.lineStyle(0);
            g.beginFill(light, 0.55);
            g.drawCircle(-s * 0.35, -s * 0.35, s * 0.3);
            g.endFill();
            break;
        }
        case 'whirl': {
            // Вихрь: маленький круглый спиннер — тело, три лопасти, ось.
            for (let i = 0; i < 3; i++) {
                const a = i * (Math.PI * 2 / 3) - Math.PI / 2;
                const cx = Math.cos(a) * s * 0.8, cy = Math.sin(a) * s * 0.8;
                g.lineStyle(1.5, 0xffffff, 0.8);
                g.beginFill(hex);
                g.drawCircle(cx, cy, s * 0.45);
                g.endFill();
                g.lineStyle(0);
                g.beginFill(light, 0.55);
                g.drawCircle(cx - s * 0.13, cy - s * 0.13, s * 0.17);
                g.endFill();
            }
            // Тело-муфта между лопастями.
            g.lineStyle(1.5, 0xffffff, 0.8);
            g.beginFill(hex);
            g.drawCircle(0, 0, s * 0.55);
            g.endFill();
            // Центральная ось.
            g.lineStyle(0);
            g.beginFill(dark, 0.85);
            g.drawCircle(0, 0, s * 0.2);
            g.endFill();
            g.lineStyle(1, light, 0.7);
            g.drawCircle(0, 0, s * 0.2);
            g.lineStyle(0);
            break;
        }
        case 'orb': {
            // Сфера: блик + тень у основания.
            g.lineStyle(1.5, 0xffffff, 0.8);
            g.beginFill(hex);
            g.drawCircle(0, 0, s);
            g.endFill();
            g.lineStyle(0);
            g.beginFill(light, 0.9);
            g.drawCircle(-s * 0.32, -s * 0.32, s * 0.36);
            g.endFill();
            g.lineStyle(0);
            g.beginFill(dark, 0.4);
            g.drawEllipse(0, s * 0.55, s * 0.6, s * 0.28);
            g.endFill();
            break;
        }
        case 'pebble': {
            // Галька: каверна + светлый верх.
            g.lineStyle(1.5, 0xffffff, 0.8);
            g.beginFill(hex);
            g.drawEllipse(0, 0, s, s * 0.72);
            g.endFill();
            g.lineStyle(0);
            g.beginFill(light, 0.35);
            g.drawEllipse(-s * 0.25, -s * 0.28, s * 0.5, s * 0.26);
            g.endFill();
            g.beginFill(dark, 0.5);
            g.drawEllipse(s * 0.15, s * 0.25, s * 0.42, s * 0.22);
            g.endFill();
            break;
        }
        case 'shard': {
            // Зазубренный кристалл-осколок с трещиной.
            g.lineStyle(1.5, 0xffffff, 0.8);
            g.beginFill(hex);
            g.moveTo(0, -s * 1.3);
            g.lineTo(s * 0.16, -s * 0.25);
            g.lineTo(s * 0.12, s * 0.55);
            g.lineTo(s * 0.42, s * 0.2);
            g.lineTo(s * 0.5, s * 0.9);
            g.lineTo(s * 0.05, s * 0.75);
            g.lineTo(-s * 0.42, s * 1.0);
            g.lineTo(-s * 0.3, s * 0.1);
            g.closePath();
            g.endFill();
            g.lineStyle(1, dark, 0.9);
            g.moveTo(-s * 0.05, -s * 1.0);
            g.lineTo(-s * 0.12, -s * 0.25);
            g.lineTo(s * 0.03, s * 0.35);
            g.lineStyle(0);
            break;
        }
        case 'cog': {
            // Шестерня: правильный профиль зубцов + ступица.
            g.lineStyle(1.5, 0xffffff, 0.8);
            g.beginFill(hex);
            const N = 8;
            g.moveTo(s * 0.8, 0);
            for (let i = 1; i < N * 2; i++) {
                const a = (i / (N * 2)) * Math.PI * 2;
                const rr = (i % 2 === 0) ? s * 0.8 : s * 0.52;
                g.lineTo(Math.cos(a) * rr, Math.sin(a) * rr);
            }
            g.closePath();
            g.endFill();
            g.lineStyle(0);
            g.beginFill(hex);
            g.drawCircle(0, 0, s * 0.42);
            g.beginHole();
            g.drawCircle(0, 0, s * 0.2);
            g.endHole();
            g.endFill();
            // Спица-светлячок на одном из зубьев (намёк на метку).
            g.beginFill(light, 0.7);
            g.drawCircle(s * 0.13, 0, s * 0.09);
            g.endFill();
            break;
        }
        case 'blade': {
            // Нож: лезвие с долом + рукоять.
            g.lineStyle(1.5, 0xffffff, 0.8);
            g.beginFill(hex);
            g.moveTo(-s * 1.4, 0);
            g.lineTo(s, -s * 0.38);
            g.lineTo(s * 0.3, -s * 0.16);
            g.lineTo(s * 0.3, s * 0.16);
            g.lineTo(s, s * 0.38);
            g.closePath();
            g.endFill();
            g.lineStyle(0);
            g.beginFill(dark);
            g.drawRoundedRect(s * 0.5, -s * 0.24, s * 0.8, s * 0.48, 2.5);
            g.endFill();
            // Дол (канавка) вдоль лезвия.
            g.lineStyle(1, dark, 0.6);
            g.moveTo(-s * 0.9, 0);
            g.lineTo(s * 0.25, 0);
            g.lineStyle(0);
            break;
        }
        case 'viper': {
            // Изогнутый клык с бликом.
            g.lineStyle(1.5, 0xffffff, 0.8);
            g.beginFill(hex);
            g.moveTo(-s * 0.6, s * 0.85);
            g.quadraticCurveTo(s * 0.35, s * 0.45, s * 0.35, -s * 0.8);
            g.lineTo(-s * 0.05, -s * 0.85);
            g.quadraticCurveTo(-s * 0.12, s * 0.2, -s * 0.85, s * 0.55);
            g.closePath();
            g.endFill();
            g.lineStyle(0);
            g.beginFill(light, 0.5);
            g.moveTo(-s * 0.68, s * 0.62);
            g.quadraticCurveTo(s * 0.02, s * 0.4, s * 0.12, -s * 0.5);
            g.lineTo(s * 0.26, -s * 0.44);
            g.quadraticCurveTo(s * 0.1, s * 0.33, -s * 0.52, s * 0.55);
            g.closePath();
            g.endFill();
            break;
        }
        case 'cube': {
            // Чёткий изометрический куб: квадратная передняя грань + верхняя и
            // правая грани (уходящие вверх-вправо), рёбра и светлый блик.
            const d = s * 0.55; // «глубина» — сдвиг задних углов вверх-вправо
            const F = { bl: [-s, s], br: [s, s], tr: [s, -s], tl: [-s, -s] };
            const B = { tr: [s + d, -s - d], tl: [-s + d, -s - d], br: [s + d, s - d] };
            g.lineStyle(0);
            g.beginFill(hex); // передняя грань
            g.moveTo(F.bl[0], F.bl[1]); g.lineTo(F.br[0], F.br[1]);
            g.lineTo(F.tr[0], F.tr[1]); g.lineTo(F.tl[0], F.tl[1]);
            g.closePath(); g.endFill();
            g.beginFill(light, 0.7); // верхняя грань
            g.moveTo(F.tl[0], F.tl[1]); g.lineTo(F.tr[0], F.tr[1]);
            g.lineTo(B.tr[0], B.tr[1]); g.lineTo(B.tl[0], B.tl[1]);
            g.closePath(); g.endFill();
            g.beginFill(dark, 0.5); // правая грань
            g.moveTo(F.br[0], F.br[1]); g.lineTo(F.tr[0], F.tr[1]);
            g.lineTo(B.tr[0], B.tr[1]); g.lineTo(B.br[0], B.br[1]);
            g.closePath(); g.endFill();
            // Рёбра и внутренние линии.
            g.lineStyle(1.5, 0xffffff, 0.8);
            g.moveTo(F.bl[0], F.bl[1]); g.lineTo(F.br[0], F.br[1]);
            g.lineTo(F.tr[0], F.tr[1]); g.lineTo(F.tl[0], F.tl[1]);
            g.closePath();
            g.lineStyle(0.8, 0xffffff, 0.5);
            g.moveTo(F.tl[0], F.tl[1]); g.lineTo(B.tl[0], B.tl[1]);
            g.moveTo(F.tr[0], F.tr[1]); g.lineTo(B.tr[0], B.tr[1]);
            g.moveTo(F.br[0], F.br[1]); g.lineTo(B.br[0], B.br[1]);
            g.lineStyle(0);
            break;
        }
        case 'bulwark': {
            // Стена с зубцами (полный профиль) и тёмной аркой-ворот.
            g.lineStyle(1.5, 0xffffff, 0.8);
            g.beginFill(hex);
            g.moveTo(-s * 0.85, s * 0.45);
            g.lineTo(s * 0.85, s * 0.45);
            g.lineTo(s * 0.85, -s * 0.45);
            g.lineTo(s * 0.75, -s * 0.45);
            g.lineTo(s * 0.75, -s * 0.9);
            g.lineTo(s * 0.45, -s * 0.9);
            g.lineTo(s * 0.45, -s * 0.45);
            g.lineTo(s * 0.15, -s * 0.45);
            g.lineTo(s * 0.15, -s * 0.9);
            g.lineTo(-s * 0.15, -s * 0.9);
            g.lineTo(-s * 0.15, -s * 0.45);
            g.lineTo(-s * 0.45, -s * 0.45);
            g.lineTo(-s * 0.45, -s * 0.9);
            g.lineTo(-s * 0.75, -s * 0.9);
            g.lineTo(-s * 0.75, -s * 0.45);
            g.lineTo(-s * 0.85, -s * 0.45);
            g.closePath();
            g.endFill();
            g.lineStyle(0);
            g.beginFill(dark, 0.85);
            g.drawRoundedRect(-s * 0.13, s * 0.02, s * 0.26, s * 0.43, 2);
            g.endFill();
            g.beginFill(light, 0.35);
            g.drawRoundedRect(-s * 0.13, -s * 0.8, s * 0.26, s * 0.3, 1);
            g.endFill();
            break;
        }
        case 'penta': {
            // Пятиугольник с внутренним бликом-контуром.
            g.lineStyle(1.5, 0xffffff, 0.8);
            g.beginFill(hex);
            petalPoly(g, 5, s * 1.05);
            g.endFill();
            g.lineStyle(1, light, 0.7);
            petalPoly(g, 5, s * 0.55);
            g.lineStyle(0);
            break;
        }
        case 'shield': {
            // Кайт-щит: перевязь + эмблема.
            g.lineStyle(1.5, 0xffffff, 0.8);
            g.beginFill(hex);
            g.moveTo(-s, -s);
            g.lineTo(s, -s);
            g.lineTo(s, s * 0.4);
            g.lineTo(0, s * 1.35);
            g.lineTo(-s, s * 0.4);
            g.closePath();
            g.endFill();
            g.lineStyle(0);
            g.beginFill(dark, 0.5);
            g.drawRoundedRect(-s * 0.72, -s * 0.66, s * 1.44, s * 0.16, 1);
            g.endFill();
            g.beginFill(light, 0.55);
            g.drawCircle(0, -s * 0.02, s * 0.26);
            g.endFill();
            break;
        }
        case 'hex': {
            // Правильный плосковершинный шестиугольник-соты: ровный контур,
            // светлый обод и затемнённая ячейка внутри (симметрично).
            const pts = (rr) => {
                const arr = [];
                for (let i = 0; i < 6; i++) {
                    const a = Math.PI / 6 + i * Math.PI / 3;
                    arr.push([Math.cos(a) * rr, Math.sin(a) * rr]);
                }
                return arr;
            };
            const poly = (arr) => {
                g.moveTo(arr[0][0], arr[0][1]);
                for (let i = 1; i < 6; i++) g.lineTo(arr[i][0], arr[i][1]);
                g.closePath();
            };
            const outer = pts(s * 1.1);
            const inner = pts(s * 0.55);
            g.lineStyle(1.5, 0xffffff, 0.8);
            g.beginFill(hex);
            poly(outer);
            g.endFill();
            // Ячейка (затемнённое ядро — эффект «соты»).
            g.lineStyle(0);
            g.beginFill(dark, 0.85);
            poly(inner);
            g.endFill();
            // Светлый обод ячейки.
            g.lineStyle(1.2, light, 0.8);
            poly(inner);
            g.lineStyle(0);
            break;
        }
        case 'hiveling': {
            // Пчела: тельце с полосками, крыльями и головой-глазком.
            g.lineStyle(1.5, 0xffffff, 0.8);
            g.beginFill(hex);
            g.drawEllipse(0, 0, s, s * 0.68);
            g.endFill();
            g.lineStyle(0);
            g.beginFill(dark, 0.85);
            g.drawEllipse(-s * 0.28, 0, s * 0.13, s * 0.58);
            g.drawEllipse(s * 0.24, 0, s * 0.13, s * 0.58);
            g.endFill();
            g.beginFill(light, 0.7);
            g.drawEllipse(-s * 0.38, -s * 0.62, s * 0.34, s * 0.22);
            g.drawEllipse(s * 0.4, -s * 0.62, s * 0.34, s * 0.22);
            g.endFill();
            g.beginFill(light);
            g.drawCircle(s * 0.62, -s * 0.18, s * 0.13);
            g.endFill();
            break;
        }
        case 'fractalite': {
            // Фрактальный треугольник (серпинский) с осколочно-тёмным ядром.
            g.lineStyle(1.5, 0xffffff, 0.8);
            g.beginFill(hex);
            g.moveTo(0, -s * 1.2);
            g.lineTo(s * 1.1, s * 0.9);
            g.lineTo(-s * 1.1, s * 0.9);
            g.closePath();
            g.endFill();
            g.lineStyle(0);
            g.beginFill(dark, 0.95);
            g.moveTo(0, -s * 0.1);
            g.lineTo(s * 0.5, s * 0.55);
            g.lineTo(-s * 0.5, s * 0.55);
            g.closePath();
            g.endFill();
            g.lineStyle(1, light, 0.5);
            g.moveTo(0, -s * 1.0);
            g.lineTo(s * 0.45, s * 0.35);
            g.lineStyle(0);
            break;
        }
        case 'spark': {
            // Четырёхлучевая звезда-искра с ядром.
            g.lineStyle(1.5, 0xffffff, 0.8);
            g.beginFill(hex);
            g.moveTo(0, -s * 1.3);
            g.lineTo(s * 0.3, -s * 0.3);
            g.lineTo(s * 1.3, 0);
            g.lineTo(s * 0.3, s * 0.3);
            g.lineTo(0, s * 1.3);
            g.lineTo(-s * 0.3, s * 0.3);
            g.lineTo(-s * 1.3, 0);
            g.lineTo(-s * 0.3, -s * 0.3);
            g.closePath();
            g.endFill();
            g.lineStyle(0);
            g.beginFill(light, 0.85);
            g.drawCircle(0, 0, s * 0.24);
            g.endFill();
            break;
        }
        case 'fragwall': {
            // Повреждённая кладка: кирпичи со швами + трещина.
            const bh = s * 0.42, br = s * 0.44, gap = s * 0.12;
            g.lineStyle(1.5, 0xffffff, 0.8);
            g.beginFill(dark);
            g.drawRoundedRect(-s * 0.8, -s * 0.55, s * 1.6, s * 1.1, 2);
            g.endFill();
            g.lineStyle(0);
            g.beginFill(hex);
            let x = -s * 0.78;
            for (let i = 0; i < 3; i++) {
                g.drawRoundedRect(x, -s * 0.5, br, bh, 1);
                x += br + gap;
            }
            x = -s * 0.5;
            for (let i = 0; i < 2; i++) {
                g.drawRoundedRect(x, s * 0.03, br, bh, 1);
                x += br + gap;
            }
            g.endFill();
            // Трещины.
            g.lineStyle(1.4, dark, 0.9);
            g.moveTo(-s * 0.7, s * 0.45);
            g.lineTo(-s * 0.2, s * 0.02);
            g.lineTo(s * 0.12, -s * 0.38);
            g.moveTo(-s * 0.15, s * 0.42);
            g.lineTo(s * 0.26, s * 0.12);
            g.lineStyle(0);
            break;
        }
        case 'void': {
            // Чёрная дыра: аккреционный диск вокруг ядра.
            g.lineStyle(1.5, 0xffffff, 0.8);
            g.beginFill(dark);
            g.drawCircle(0, 0, s);
            g.endFill();
            g.lineStyle(1.6, light, 0.85);
            g.drawCircle(0, 0, s * 0.62);
            g.lineStyle(1, light, 0.45);
            g.drawCircle(0, 0, s * 0.8);
            g.lineStyle(0);
            g.beginFill(0x000000, 0.9);
            g.drawCircle(0, 0, s * 0.42);
            g.endFill();
            break;
        }
        case 'entropy': {
            // Ядро хаоса: тёмная ступица с расходящимися спицами-осколками.
            g.lineStyle(0);
            g.beginFill(dark);
            g.drawCircle(0, 0, s * 0.55);
            g.endFill();
            for (let i = 0; i < 6; i++) {
                const a = (i / 6) * Math.PI * 2;
                g.lineStyle(2.2, hex, 1);
                g.moveTo(Math.cos(a) * s * 0.58, Math.sin(a) * s * 0.58);
                g.lineTo(Math.cos(a) * s * 0.98, Math.sin(a) * s * 0.98);
                g.lineStyle(0);
                g.beginFill(hex);
                g.drawCircle(Math.cos(a) * s * 0.98, Math.sin(a) * s * 0.98, s * 0.13);
                g.endFill();
            }
            g.beginFill(light);
            g.drawCircle(0, 0, s * 0.17);
            g.endFill();
            break;
        }
        case 'singularity': {
            const tt = performance.now() / 1000;
            // Singularity: тёмное вращающееся ядро с вихревыми спицами.
            g.lineStyle(0);
            g.beginFill(0x0a0618);
            g.drawCircle(0, 0, s * 0.55);
            g.endFill();
            for (let i = 0; i < 6; i++) {
                const a = (i / 6) * Math.PI * 2 + tt * 0.8;
                g.lineStyle(2.5, dark, 1);
                g.moveTo(Math.cos(a) * s * 0.58, Math.sin(a) * s * 0.58);
                g.lineTo(Math.cos(a) * s * 0.98, Math.sin(a) * s * 0.98);
                g.lineStyle(0);
                g.beginFill(dark);
                g.drawCircle(Math.cos(a) * s * 0.98, Math.sin(a) * s * 0.98, s * 0.13);
                g.endFill();
            }
            g.beginFill(light);
            g.drawCircle(0, 0, s * 0.17);
            g.endFill();
            break;
        }
        case 'tear': {
            // Слеза: капля с бликом, остриём вверх.
            g.lineStyle(1.5, 0xffffff, 0.8);
            g.beginFill(hex);
            g.moveTo(0, -s * 1.25);
            g.quadraticCurveTo(s * 0.95, -s * 0.15, s * 0.58, s * 0.5);
            g.quadraticCurveTo(s * 0.42, s * 1.05, 0, s * 0.9);
            g.quadraticCurveTo(-s * 0.42, s * 1.05, -s * 0.58, s * 0.5);
            g.quadraticCurveTo(-s * 0.95, -s * 0.15, 0, -s * 1.25);
            g.endFill();
            g.lineStyle(0);
            g.beginFill(light, 0.55);
            g.drawEllipse(-s * 0.3, -s * 0.52, s * 0.2, s * 0.32);
            g.endFill();
            break;
        }
        case 'line': {
            // Антенна-обзор: вертикальная мачта с сенсором-шаром сверху (видна в
            // центре игрока, как «прибор»).
            g.lineStyle(1.2, 0xffffff, 0.9);
            g.moveTo(0, s * 0.7);
            g.lineTo(0, -s * 1.1);
            g.lineStyle(0);
            g.beginFill(hex);
            g.drawCircle(0, -s * 1.25, s * 0.35);
            g.endFill();
            g.beginFill(light, 0.7);
            g.drawCircle(-s * 0.12, -s * 1.32, s * 0.12);
            g.endFill();
            // База-плашка.
            g.lineStyle(1.2, dark, 0.8);
            g.moveTo(-s * 0.35, s * 0.7);
            g.lineTo(s * 0.35, s * 0.7);
            g.lineStyle(0);
            break;
        }
        case 'compass': {
            // Компас: кружок с розой ветров (стрелка поверх — в renderPetals).
            g.lineStyle(1.5, 0xffffff, 0.8);
            g.beginFill(hex);
            g.drawCircle(0, 0, s);
            g.endFill();
            g.lineStyle(0);
            g.beginFill(dark, 0.9);
            g.drawCircle(0, 0, s * 0.6);
            g.endFill();
            g.beginFill(light, 0.9);
            for (let i = 0; i < 4; i++) {
                const a = i * Math.PI / 2;
                g.drawCircle(Math.cos(a) * s * 0.42, Math.sin(a) * s * 0.42, s * 0.09);
            }
            g.endFill();
            // Центральная ось (по ней «ползёт» стрелка).
            g.lineStyle(1, light, 0.5);
            g.moveTo(-s * 0.3, 0);
            g.lineTo(s * 0.3, 0);
            g.lineStyle(0);
            break;
        }
        case 'saturn': {
            // Сатурн: планета с наклонным кольцом (задняя полоса за телом,
            // передняя — дугой поверх снизу).
            g.lineStyle(0);
            g.beginFill(dark, 0.85);
            g.drawEllipse(0, -s * 0.05, s * 1.4, s * 0.45);
            g.endFill();
            g.beginFill(hex);
            g.drawCircle(0, 0, s * 0.78);
            g.endFill();
            g.beginFill(light, 0.5);
            g.drawEllipse(-s * 0.18, -s * 0.28, s * 0.32, s * 0.18);
            g.endFill();
            g.beginFill(dark, 0.45);
            g.drawEllipse(0, s * 0.5, s * 0.46, s * 0.18);
            g.endFill();
            g.lineStyle(1.4, light, 0.85);
            g.moveTo(-s * 1.4, -s * 0.08);
            g.quadraticCurveTo(0, s * 0.55, s * 1.4, -s * 0.08);
            g.lineStyle(0);
            break;
        }
        case 'rune': {
            // Руна: гранёная пластина с руническими штрихами (ᚱ-подобный знак).
            g.lineStyle(1.5, 0xffffff, 0.8);
            g.beginFill(hex);
            petalPoly(g, 6, s * 1.05);
            g.endFill();
            g.lineStyle(0);
            g.beginFill(dark, 0.9);
            petalPoly(g, 6, s * 0.62);
            g.endFill();
            g.lineStyle(1.6, light, 0.95);
            g.moveTo(-s * 0.18, -s * 0.5);
            g.lineTo(s * 0.18, -s * 0.5);
            g.lineTo(s * 0.18, s * 0.42);
            g.moveTo(s * 0.18, s * 0.42);
            g.lineTo(-s * 0.24, s * 0.14);
            g.moveTo(-s * 0.24, s * 0.14);
            g.lineTo(-s * 0.05, -s * 0.5);
            g.lineStyle(0);
            break;
        }
        case 'eye': {
            // Глаз: миндалевидное веко, радужка и зрачок.
            g.lineStyle(1.5, 0xffffff, 0.8);
            g.beginFill(hex);
            g.moveTo(0, -s * 0.95);
            g.quadraticCurveTo(s * 1.2, 0, 0, s * 0.95);
            g.quadraticCurveTo(-s * 1.2, 0, 0, -s * 0.95);
            g.endFill();
            g.lineStyle(0);
            g.beginFill(0xffffff, 0.95);
            g.drawEllipse(0, 0, s * 0.62, s * 0.5);
            g.endFill();
            g.beginFill(hex, 0.95);
            g.drawCircle(0, 0, s * 0.3);
            g.endFill();
            g.beginFill(0x000000, 0.95);
            g.drawCircle(0, 0, s * 0.14);
            g.endFill();
            g.beginFill(light, 0.85);
            g.drawCircle(-s * 0.12, -s * 0.16, s * 0.08);
            g.endFill();
            break;
        }
        case 'spike': {
            // Шипастое кольцо: ободок с острыми шипами наружу (носится на игроке).
            g.lineStyle(1.5, 0xffffff, 0.8);
            g.beginFill(hex);
            const spikes = 12;
            for (let i = 0; i < spikes; i++) {
                const a = (i / spikes) * Math.PI * 2;
                const ax = Math.cos(a), ay = Math.sin(a);
                const rx = Math.cos(a + Math.PI / spikes), ry = Math.sin(a + Math.PI / spikes);
                // Остриё шипа (длинное) + впадина между шипами (короткая).
                g.moveTo(ax * s, ay * s);
                g.lineTo(rx * s * 1.5, ry * s * 1.5);
                g.lineTo(rx * s, ry * s);
            }
            g.endFill();
            g.lineStyle(0);
            g.beginFill(dark, 0.9);
            g.drawCircle(0, 0, s * 0.42);
            g.endFill();
            g.beginFill(light, 0.85);
            g.drawCircle(0, 0, s * 0.17);
            g.endFill();
            break;
        }
        default: {
            // Новые виды (volt/jag/anvil/lance/saw/flare/piercer/wingblade/
            // rivet/tail/trap/жетоны...) до появления авторских дизайнов
            // (этап 7): фирменный круг-заглушка с бликом.
            g.lineStyle(1.5, petalShade(hex, 0.6), 0.95);
            g.beginFill(hex);
            g.drawCircle(0, 0, s);
            g.endFill();
            g.lineStyle(0);
            g.beginFill(0xffffff, 0.35);
            g.drawEllipse(-s * 0.25, -s * 0.35, s * 0.55, s * 0.3);
            g.endFill();
            break;
        }
    }
}

// ----- Отрисовка петалей-предметов на земле -----
function renderGround(ground) {
    const seen = new Set();
    for (const g of ground) {
        seen.add(g.id);
        let a = groundActors[g.id];
        if (!a) {
            const c = new PIXI.Container();
            const spr = new PIXI.Graphics();
            drawPetalShape(spr, g);
            c.addChild(spr);
            // Подпись редкости под петалей.
            const txt = new PIXI.Text(g.rarity, {
                fontFamily: 'Arial', fontSize: 9, fill: '#ffffff',
                stroke: '#000000', strokeThickness: 2,
            });
            txt.anchor.set(0.5);
            txt.y = g.size + 9;
            c.addChild(txt);
            // Свечение дропа редкости Fractal (выпадает только с Fractal/Chaos-
            // боссов): фиолетовый пульсирующий ореол + мерцающие блики-звёздочки
            // (настоящее «сверкание»). Рисуется каждый кадр в tick().
            let glow = null;
            let sparkles = [];
            // Свечение дропов HIGH-редкостей: Fractal (6) — фиолетовый ореол,
            // Hexagon (5) — янтарный мини-босс разблеск.
            if (g.rarityI === 6 || g.rarityI === 5) {
                glow = new PIXI.Graphics();
                c.addChildAt(glow, 0);
                const sz = g.size || 8;
                for (let i = 0; i < 9; i++) {
                    sparkles.push({
                        ang: Math.random() * 6.283,
                        r: (1.2 + Math.random() * 0.5) * sz,
                        ph: Math.random() * 6.283,
                        sp: 1.4 + Math.random() * 1.4,
                        w: Math.random() < 0.5,
                    });
                }
            }
            c.position.set(g.x, g.y);
            world.addChild(c);
            c.alpha = 0; // фейд-ин
            a = { c, spr, glow, sp: sparkles, glowRar: g.rarityI, size: g.size || 8, targetX: g.x, targetY: g.y, bob: Math.random() * 6.28 };
            groundActors[g.id] = a;
        }
        a.targetX = g.x;
        a.targetY = g.y;
        // Фейд-ин нового дропа на земле.
        if (a.c.alpha < 1) a.c.alpha = Math.min(1, a.c.alpha + 0.14);
    }
    for (const id of Object.keys(groundActors)) {
        if (!seen.has(id)) {
            const a = groundActors[id];
            // Петаль-дроп подобрали (или истёк таймер): не исчезаем мгновенно,
            // а плавно «взлетаем» и растворяемся (анимация в tick).
            groundFades.set(id, { c: a.c, t: 0 });
            delete groundActors[id];
        }
    }
}

// renderProjectiles — летящие снаряды (миссил-blade, осколки-hiveling).
function renderProjectiles(projs) {
    const seen = new Set();
    for (const p of projs) {
        seen.add(p.id);
        let a = projectileActors[p.id];
        if (!a) {
            const c = new PIXI.Container();
            const spr = new PIXI.Graphics();
            drawProjectileShape(spr, p.kind, p.size, p.color);
            c.addChild(spr);
            c.position.set(p.x, p.y);
            world.addChild(c);
            a = { c, spr, targetX: p.x, targetY: p.y, rot: p.angle };
            projectileActors[p.id] = a;
        }
        a.targetX = p.x;
        a.targetY = p.y;
        a.rot = p.angle;
    }
    for (const id of Object.keys(projectileActors)) {
        if (!seen.has(id)) {
            world.removeChild(projectileActors[id].c);
            delete projectileActors[id];
        }
    }
}

// drawProjectileShape рисует снаряд, ориентированный вдоль +X (поворот — в tick).
// Добавлено мягкое свечение, чтобы летящий клинок/осколок был хорошо виден.
function drawProjectileShape(g, kind, size, color) {    g.clear();
    // Свечение позади снаряда.
    g.beginFill(0xffffff, 0.18);
    g.drawCircle(0, 0, size * 1.5);
    g.endFill();
    g.beginFill(color, 0.35);
    g.drawCircle(0, 0, size * 1.0);
    g.endFill();
    // Сам снаряд.
    g.lineStyle(1.5, 0xffffff, 0.9);
    g.beginFill(color);
    if (kind === 'blade') {
        // Удлинённый клинок, летящий «остриём вперёд».
        g.moveTo(size, 0);
        g.lineTo(-size * 0.35, size * 0.3);
        g.lineTo(-size * 0.7, 0);
        g.lineTo(-size * 0.35, -size * 0.3);
        g.closePath();
    } else if (kind === 'beamorb') {
        // Энергетический шар Beam: яркое ядро в цветной оболочке.
        g.drawCircle(0, 0, size * 0.72);
        g.lineStyle(0);
        g.beginFill(0xffffff, 0.95);
        g.drawCircle(0, 0, size * 0.38);
        g.endFill();
        g.beginFill(color, 0.55);
        g.drawCircle(0, 0, size * 0.55);
        g.endFill();
    } else {
        // Шарик (разрыв hiveling — овальный петаль).
        g.drawCircle(0, 0, size * 0.7);
    }
    g.endFill();
}

// renderChains — визуальные линии-цепи петеля spark: настоящая зигзаг-молния
// (3 слоя: мягкое свечение → средней толщины стек → белое ядро), с дрожанием
// и вспышками на контактах. Рисуется ПОВЕРХ ВСЕГО (chainGfx поднимается в
// топ world каждый кадр). Исчезает, когда сервер перестаёт слать.
let chainGfx = null;
function renderChains(chains) {
    if (!chainGfx) {
        chainGfx = new PIXI.Graphics();
        world.addChild(chainGfx);
    }
    if (chainGfx.parent === world) world.addChild(chainGfx); // поверх всего
    chainGfx.clear();
    const col = 0xffd75e, colLight = 0xfff3c4;
    for (const ch of chains || []) {
        const pts = ch.points || [];
        if (pts.length < 2) continue;
        // Зигзаг между подряд идущими точками цепи (дрожит каждый кадр).
        const zig = [];
        zig.push([pts[0].x, pts[0].y]);
        for (let i = 1; i < pts.length; i++) {
            const ax = pts[i - 1].x, ay = pts[i - 1].y, bx = pts[i].x, by = pts[i].y;
            const dx = bx - ax, dy = by - ay;
            const len = Math.hypot(dx, dy) || 1;
            const nx = -dy / len, ny = dx / len;
            const segs = Math.max(3, Math.min(10, Math.floor(len / 55)));
            for (let k = 1; k < segs; k++) {
                const f = k / segs;
                const amp = Math.sin(f * Math.PI) * len * 0.14 * (Math.random() < 0.5 ? -1 : 1);
                zig.push([ax + dx * f + nx * amp, ay + dy * f + ny * amp]);
            }
            zig.push([bx, by]);
        }
        if (zig.length < 2) continue;
        const stroke = (w, c, a) => {
            chainGfx.lineStyle(w, c, a);
            chainGfx.moveTo(zig[0][0], zig[0][1]);
            for (let i = 1; i < zig.length; i++) chainGfx.lineTo(zig[i][0], zig[i][1]);
            chainGfx.lineStyle(0);
        };
        stroke(10, col, 0.16);   // мягкое свечение
        stroke(5, colLight, 0.5); // средний слой
        stroke(2.2, 0xffffff, 0.95); // белое ядро
        // Вспышки на точках стыка (начало/каждый задетый моб).
        for (const p of pts) {
            const rr = 7 + Math.random() * 4;
            chainGfx.lineStyle(2, colLight, 0.85);
            chainGfx.beginFill(0xffffff, 0.75);
            chainGfx.drawCircle(p.x, p.y, rr * 0.55);
            chainGfx.endFill();
            chainGfx.beginFill(colLight, 0.25);
            chainGfx.drawCircle(p.x, p.y, rr);
            chainGfx.endFill();
            chainGfx.lineStyle(0);
        }
    }
}

// ----- Экран смерти -----
function showDeath() {
    updateHP(0, lastMaxHP);
    deathEl.style.display = 'flex';
    // Мой игрок полностью исчезает (вместе с орбитальными петалями).
    if (myId && actors[myId]) actors[myId].sprite.visible = false;
    // Волновой режим: ручного респавна нет — ждём начала следующей волны.
    const respawnBtnEl = document.getElementById('respawnBtn');
    const deathSub = document.getElementById('deathSub');
    if (waveModeClient) {
        if (respawnBtnEl) respawnBtnEl.style.display = 'none';
        if (deathSub) {
            deathSub.textContent = 'You will respawn at the start of the next wave';
            deathSub.style.display = 'block';
        }
    } else {
        if (respawnBtnEl) respawnBtnEl.style.display = '';
        if (deathSub) deathSub.style.display = 'none';
    }
    syncTicker();
}

function hideDeath() {
    deathEl.style.display = 'none';
    if (myId && actors[myId]) actors[myId].sprite.visible = true;
    syncTicker();
}

// ----- Чат -----
const chatToastEl = document.getElementById('chatToast');
const chatToastMs = 8000; // сколько виден тост, пока не открыт чат

// Показывает всплывающий тост-уведомление. Исчезает через chatToastMs,
// историю смотреть можно, открыв чат.
function pushToast(className, text) {
    if (!chatToastEl) return;
    // Пока открыт полный чат — тосты не показываем (не накладываемся на лог).
    if (chatOpen) return;
    const t = document.createElement('div');
    t.className = 'chat-toast ' + className;
    t.innerHTML = text;
    chatToastEl.appendChild(t);
    // Ограничиваем число видимых тостов — прячем самые старые.
    while (chatToastEl.children.length > 4) {
        const old = chatToastEl.firstElementChild;
        if (old) old.remove();
    }
    setTimeout(() => {
        t.classList.add('hidden');
        setTimeout(() => { if (t.parentNode) t.remove(); }, 350);
    }, chatToastMs);
}

function addSystemToast(text, kind) {
    // Показываем тост-уведомление + фиксируем в логе чата.
    // super — розовое (Super-моб); announce — золотое объявление (админ);
    // craftFractal/craftChaos — сияющее объявление о крафте высокого тира
    // с цветом редкости: Fractal — фиолетовый, Chaos — розово-магентовый.
    let cls = 'sys';
    if (kind === 'announce') cls = 'sys announce';
    else if (kind === 'craftFractal') cls = 'sys craft fractal';
    else if (kind === 'craftChaos') cls = 'sys craft chaos';
    pushToast(cls, escapeHtml(text));
    const row = document.createElement('div');
    row.className = 'msg ' + cls;
    const tx = document.createElement('span');
    tx.textContent = text;
    row.appendChild(tx);
    addChatLogRow(row);
}

// Постоянный лог чата: сообщения ОСТАЮТСЯ (как раньше), старые не удаляются.
// Все новые фичи (зоны, ники, цвета, тосты, пузыри) сохранены. Чтобы лог не
// рос бесконечно, держим максимум 80 строк — самые старые отрезаются.
const CHAT_LOG_MAX = 80;
function addChatLogRow(row) {
    // Если активен поиск — новые строки не показываем, если не совпадают.
    if (chatSearchQuery) {
        row.style.display = rowMatchesQuery(row, chatSearchQuery) ? '' : 'none';
    }
    chatLog.appendChild(row);
    while (chatLog.children.length > CHAT_LOG_MAX) {
        if (chatLog.firstChild) chatLog.removeChild(chatLog.firstChild);
    }
    chatLog.scrollTop = chatLog.scrollHeight;
}

// Фильтр-поиск по истории чата: скрывает строки, не содержащие запрос.
let chatSearchQuery = '';
function chatLogFilterUpdate() {
    const q = chatSearchQuery.toLowerCase();
    for (const row of chatLog.children) {
        row.style.display = (!q || rowMatchesQuery(row, q)) ? '' : 'none';
    }
    chatLog.scrollTop = chatLog.scrollHeight;
}
function rowMatchesQuery(row, q) {
    // Ищем подстроку во всём тексте строки (без учёта регистра).
    return (row.textContent || '').toLowerCase().includes(q);
}

// Поисковая строка чата: фильтрует историю на лету.
(function () {
    const inp = document.getElementById('chatSearch');
    if (!inp) return;
    inp.addEventListener('input', () => {
        chatSearchQuery = inp.value.trim();
        chatLogFilterUpdate();
    });
    // Не отправляем сообщения из поиска; Enter — просто снимаем фокус.
    inp.addEventListener('keydown', (e) => {
        e.stopPropagation();
    });
})();

function addChatMessage(name, text, zone, senderId) {
    const row = document.createElement('div');
    row.className = 'msg ' + (name === myNickname ? 'me' : 'other');
    if (zone) {
        const zn = document.createElement('span');
        zn.className = 'zone-tag';
        const zc = zoneTagColor(zone);
        if (zc) zn.style.color = zc;
        zn.textContent = '[' + zone + '] ';
        row.appendChild(zn);
    }
    const nm = document.createElement('span');
    nm.className = 'name';
    nm.textContent = name + ': ';
    const tx = document.createElement('span');
    tx.textContent = text;
    row.appendChild(nm);
    row.appendChild(tx);
    addChatLogRow(row);

    // Тост виден только когда чат закрыт (иначе и так видно в логе).
    // В тосте всегда реальный ник (без «You») и зона отправителя.
    if (!chatOpen) {
        const me = name === myNickname;
        const zc = zoneTagColor(zone);
        const ztag = zone
            ? `<span class="zone-tag"${zc ? ` style="color:${zc}"` : ''}>[${escapeHtml(zone)}]</span> `
            : '';
        pushToast(me ? 'me' : 'other',
            `${ztag}<span class="name">${escapeHtml(name)}:</span> ${escapeHtml(text)}`);
    }
    // Пузырь речи над игроком в мире: живёт столько же, сколько тост.
    showChatBubble(senderId, text, zone);
}

// Речевой пузырь над игроком: пока «живо» исчезающее сообщение чата, текст
// висит над головой отправителя и виден всем игрокам рядом.
function showChatBubble(playerId, text, zone) {
    const actor = actors[playerId];
    if (!actor) return;
    if (actor.chatBubbleTimer) clearTimeout(actor.chatBubbleTimer);
    if (actor.chatBubble && actor.chatBubble.parent) actor.chatBubble.parent.removeChild(actor.chatBubble);
    // Красивая рамочка со скруглёнными углами: тёмная пилюля со стеклянным
    // глянцем и светлой обводкой, текст ПО ЦЕНТРУ пилюли.
    const t = new PIXI.Text(text, {
        fontFamily: 'Arial', fontSize: 12, fontWeight: 'bold', fill: '#ffffff',
        stroke: '#000000', strokeThickness: 3,
        wordWrap: true, wordWrapWidth: 190, breakWords: true,
        align: 'center',
    });
    t.anchor.set(0.5, 0.5);
    const padX = 9, padY = 6;
    const bw = Math.max(t.width, 36) + padX * 2;
    const bh = t.height + padY * 2;
    const pill = new PIXI.Graphics();
    pill.beginFill(0x0e1118, 0.88);
    pill.lineStyle(2, 0x7be0ff, 0.55);
    pill.drawRoundedRect(-bw / 2, -bh / 2, bw, bh, 12);
    pill.endFill();
    pill.lineStyle(0);
    pill.beginFill(0xffffff, 0.07);
    pill.drawRoundedRect(-bw / 2 + 2, -bh / 2 + 2, bw - 4, (bh - 4) * 0.45, 10);
    pill.endFill();
    const c = new PIXI.Container();
    c.addChild(pill);
    c.addChild(t);
    // Пузырь живёт в ОТДЕЛЬНОМ верхнем слое (поверх петалей и мобов),
    // позиция синхронизируется с игроком каждый кадр.
    c.actorId = playerId;
    c.offY = -(actor.drawnSize || 15) - 42;
    ensureBubbleLayer().addChild(c);
    c.position.set(actor.sprite.x, actor.sprite.y + c.offY);
    actor.chatBubble = c;
    actor.chatBubbleTimer = setTimeout(() => {
        if (!c.parent) return;
        c.alpha = 0;
        setTimeout(() => { if (c.parent) c.parent.removeChild(c); }, 300);
        if (actor.chatBubble === c) { actor.chatBubble = null; actor.chatBubbleTimer = null; }
    }, chatToastMs);
}

// Слой пузырей чата: держится последним ребёнком world — всегда поверх петалей.
let chatBubbleLayer = null;
function ensureBubbleLayer() {
    if (!chatBubbleLayer) {
        chatBubbleLayer = new PIXI.Container();
        world.addChild(chatBubbleLayer);
    }
    if (world.children[world.children.length - 1] !== chatBubbleLayer) {
        world.addChild(chatBubbleLayer);
    }
    return chatBubbleLayer;
}

function escapeHtml(s) {
    return String(s).replace(/[&<>"']/g, c => ({
        '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
    }[c]));
}

// ----- Тултип петали: название, редкость, способность, свойства, HP, DMG -----
// RAR_MULT — из единого реестра js/rarities.js (автогенерация из server/rarity,
// 16 редкостей; фолбэк, если сервер не прислал hp/dmg).
// Константы кольца-ауры Saturn (дублируют server/mob.go). Только визуал —
// урон считается на сервере.
const SAT_RING_BASE = 100;
const SAT_RING_PER_RAR = 20;
const SAT_RING_THICK = 26;
const PETAL_DB = {
    // cd = время полной ПЕРЕЗАРЯДКИ петеля в секундах: 5.0s (база) — после
    // поломки петаль исчезает и через это время возвращается на орбиту.
    // cd: 0 — аксессуары (eye/line/spike): никогда не ломаются, без перезарядки.
    oval:       { hp: 8,   dmg: 4.5,  cd: 4.0, abl: 'Lifesteal', desc: 'Heals you 30% of the damage it deals.' },
    thorns:     { hp: 5,   dmg: 7.5,  cd: 4.5, abl: 'Thorns',    desc: 'Reflects 40% of the mob\'s hit back at it.' },
    rock:       { hp: 14,  dmg: 2.5,  cd: 5.0, abl: 'Armor',     desc: 'Takes 50% less wear from mob hits.' },
    ball:       { hp: 6,   dmg: 3,    cd: 5.0, abl: 'Bomb',      desc: 'On hit: explodes for 50% DMG to nearby mobs (radius 80).' },
    orb:        { hp: 7,   dmg: 4,    cd: 5.0, abl: 'Regen',     desc: 'Passive. Regen 3% MaxHP/s on orbit. Only the best orb counts.' },
    pebble:     { hp: 9,   dmg: 3,    cd: 3.0, abl: 'Sling',     desc: 'Very fast hits with a strong knockback.' },
    shard:      { hp: 5.5, dmg: 10,   cd: 4.5, abl: 'Wound',     desc: 'On hit: 1s wound for 50% DMG/s.' },
    cog:        { hp: 10,  dmg: 4.5,  cd: 5.0, abl: 'Overdrive', desc: 'Passive. Speeds up all petal recharges (stacks).' },
    whirl:      { hp: 10,  dmg: 4.5,  cd: 5.0, abl: 'Spin',      desc: 'Passive. Orbit spin +30%, +10%/rarity (stacks).' },
    blade:      { hp: 8,   dmg: 11,   cd: 5.0, abl: 'Missile',   desc: 'Space: fires a piercing missile (flies out, returns in 1.2s).' },
    viper:      { hp: 1,   dmg: 8,    cd: 7.5, abl: 'Poison',    desc: 'On hit: poison 3.5s, ×10 DMG/s. One poison per mob (no stack). Weak (×0.35) on higher tiers.' },
    cube:       { hp: 15,  dmg: 2,    cd: 5.0, abl: 'Push',      desc: 'Heavy knockback on every hit, low damage.' },
    bulwark:    { hp: 26,  dmg: 2,    cd: 5.0, abl: 'MaxHP',     desc: 'Passive. +80% base MaxHP ×rarity mult per petal (stacks).' },
    penta:      { hp: 9,   dmg: 6.5,  cd: 5.0, abl: 'Swift',     desc: 'Passive. +10% move speed, +2%/rarity (stacks).' },
    shield:     { hp: 13,  dmg: 6,    cd: 5.0, abl: 'Shield',    desc: 'Passive. Charges a shield that absorbs damage: +5/s, cap 10 (×rarity).' },
    hex:        { hp: 10,  dmg: 7,    cd: 5.0, abl: 'Magnet',    desc: 'Passive. Magnet: pulls drops from 250 + 45×rarity range.' },
    hiveling:   { hp: 1,   dmg: 9,    cd: 5.0, abl: 'Split',     desc: 'On destroy: bursts into 5 piercing shards (60% DMG each).' },
    fractalite: { hp: 1,   dmg: 15,   cd: 5.0, abl: 'Twin',      desc: '40% chance to strike twice.' },
    spark:      { hp: 4.67, dmg: 8,    cd: 5.0, abl: 'Chain',     desc: 'Chains to 6 mobs (range 320), −25% damage per link.' },
    fragwall:   { hp: 19,  dmg: 1,    cd: 5.0, abl: 'Guard',     desc: 'Passive. Incoming damage −5% (−2%/rarity) per petal; stacks, min 20% lands.' },
    void:       { hp: 11,  dmg: 12,   cd: 5.0, abl: 'Dash',      desc: 'Space: dash 180 units in the move direction, 3s recharge.' },
    entropy:    { hp: 8,   dmg: 0,    cd: 5.0, abl: 'Minions',   desc: 'Summons 2 friendly mobs (respawn 7s; Fractal — 12s). Stats = real mob stats of that rarity; Pentagon+ — MK (×2/×5).' },
    singularity:{ hp: 10,  dmg: 0,    cd: 5.0, abl: 'Minions',   desc: 'Summons 2 Black Hole minions (respawn 7s; Fractal — 12s). Stats = real mob stats of that rarity; Pentagon+ — MK (×2/×5).' },
    tear:       { hp: 9,   dmg: 3,    cd: 5.0, abl: 'Tear',      desc: 'When broken: heals you 15% MaxHP (+7.5%/rarity).' },
    line:       { hp: 10,  dmg: 0,    cd: 0,    abl: 'View',      desc: 'Accessory. View radius +10%/rarity (never breaks).' },
    compass:    { hp: 14,  dmg: 1,    cd: 5.0, abl: 'Compass',   desc: 'Points to the nearest Fractal/Chaos boss (deep zones only).' },
    saturn:     { hp: 22,  dmg: 7,    cd: 5.0, abl: 'Rings',     desc: 'Damage aura around you (radius 100 + 20×rarity). One per loadout.' },
    rune:       { hp: 13,  dmg: 7,    cd: 5.0, abl: 'Rune',      desc: '3 hits on a mob → explosion: 2× DMG AoE (radius 130).' },
    eye:        { hp: 14,  dmg: 0,    cd: 0,    abl: 'Eye',       desc: 'Accessory. Widens the petal orbit +10%/rarity (never breaks).' },
    spike:      { hp: 10,  dmg: 0,    cd: 0,    abl: 'Body',      desc: 'Accessory. Body contact deals 20×rarity damage per spike (0.5s tick, stacks).' },
    rhombus:    { hp: 14,  dmg: 6,    cd: 5.0,  abl: 'Slow',      desc: 'On hit: slows the mob 45% for 3s (same or lower rarity only).' },
    twirl:      { hp: 10,  dmg: 6,    cd: 6.0,  abl: 'Reverse',   desc: 'Reverses the orbit rotation direction (each petal flips it back).' },
    // Новые ДПС-петали (порт worlddata, этап 3): cd = 5.0с × множитель вида.
    volt:       { hp: 4,   dmg: 11,   cd: 5.0,  abl: 'Volt',       desc: 'Pure DPS petal (drops from Comet).' },
    jag:        { hp: 3.5, dmg: 9.5,  cd: 3.5,  abl: 'Jag',        desc: 'Fast jagged DPS (drops from Zigzag).' },
    anvil:      { hp: 12,  dmg: 10,   cd: 6.5,  abl: 'Anvil',      desc: 'Heavy DPS tank (drops from Quarry).' },
    lance:      { hp: 4,   dmg: 13,   cd: 5.0,  abl: 'Lance',      desc: 'Glass cannon: highest raw damage.' },
    saw:        { hp: 6.5, dmg: 9.5,  cd: 4.5,  abl: 'Saw',        desc: 'Balanced DPS.' },
    flare:      { hp: 9,   dmg: 8.5,  cd: 5.0,  abl: 'Flare',      desc: 'Durable DPS for long fights.' },
    piercer:    { hp: 4,   dmg: 12,   cd: 5.0,  abl: 'Piercer',    desc: 'Sharp wedge, high burst (drops from Arrowhead).' },
    wingblade:  { hp: 5,   dmg: 10,   cd: 4.0,  abl: 'Wingblade',  desc: 'Fast flying shard (drops from Shardwing).' },
    rivet:      { hp: 20,  dmg: 3,    cd: 5.0,  abl: 'Armor',      desc: 'Armored plate, very durable (drops from Frame).' },
    tail:       { hp: 6,   dmg: 7,    cd: 3.5,  abl: 'Push',       desc: 'Whip tail with knockback (drops from Kite).' },
    trap:       { hp: 10,  dmg: 0,    cd: 6.0,  abl: 'Trap',       desc: 'Soulbound capture gift. Not craftable, not absorbable.' },
    yang:       { hp: 1,   dmg: 0,    cd: 0,    abl: 'Token',      desc: 'Story token. Never fights, never breaks.' },
    ying:       { hp: 1,   dmg: 0,    cd: 0,    abl: 'Token',      desc: 'Story token. Never fights, never breaks.' },
    yinyang:    { hp: 1,   dmg: 0,    cd: 0,    abl: 'Token',      desc: 'Story token. Never fights, never breaks.' },
    nexus:      { hp: 10,  dmg: 0,    cd: 0,    abl: 'Summon',     desc: 'Admin petal: summons a friendly Nexus.' },
};

const petalTipEl = document.getElementById('petalTip');
// Грейды (порт server/pot.go): лестница UNIQUE..PARAGON, подпись по Pot.
const GRADE_PCT = [100, 110, 125, 150, 175, 200];
const GRADE_NAME = ['', 'UNIQUE', 'SUPERB', 'PRISTINE', 'MYTHIC', 'PARAGON'];
function gradeLabelOf(pot) {
    const p = (pot == null || pot < 100) ? 100 : pot;
    let best = 0, bd = -1;
    for (let i = 0; i < GRADE_PCT.length; i++) {
        let d = Math.abs(p - GRADE_PCT[i]);
        if (bd < 0 || d < bd) { best = i; bd = d; }
    }
    return best > 0 ? GRADE_NAME[best] + ' ×' + GRADE_PCT[best] + '%' : '';
}
function petalTipHTML(def) {
    const kind = def.kind || 'oval';
    const rar = def.rarity || 0;
    const db = PETAL_DB[kind] || { hp: 8, dmg: 4, cd: '', abl: '', desc: '' };
    const mult = RAR_MULT[Math.min(rar, RAR_MULT.length - 1)];
    const col = rarityColors[Math.min(rar, rarityColors.length - 1)];
    const hp = Math.round((def.hp != null ? def.hp : db.hp * mult) * 10) / 10;
    const dmg = Math.round((def.dmg != null ? def.dmg : db.dmg * mult) * 10) / 10;
    // Recharge = время полной ПЕРЕЗАРЯДКИ петеля в секундах: после поломки
    // петаль исчезает и через это время возвращается на орбиту. База — 5.0с
    // × множитель вида (порт родителя; аксессуары никогда не ломаются).
    let cdTxt = 'None (never breaks)';
    if (db.cd > 0) cdTxt = db.cd + 's';
    // Грейд (Pot) и провенанс крафта — серверные поля, клиент только показывает.
    const grade = gradeLabelOf(def.pot);
    const crafted = def.craftedBy ? '<div class="t-desc">Crafted by ' + escapeHtml(def.craftedBy) + '</div>' : '';
    return '<div class="t-name" style="color:' + col + '">' + escapeHtml(def.name || kindName(kind)) + '</div>' +
        '<div class="t-rar">Rarity: <b style="color:' + col + '">' + rarityName(rar) + '</b> (' + rar + ')</div>' +
        (grade ? '<div class="t-abl" style="color:#ffd54f">★ ' + grade + '</div>' : '') +
        (db.abl ? '<div class="t-abl">✦ ' + db.abl + '</div><div class="t-desc">' + escapeHtml(db.desc) + '</div>' : '') +
        '<div class="t-stats">HP: <b>' + hp + '</b> · DMG: <b>' + dmg + '</b> · Recharge: <b>' + cdTxt + '</b></div>' +
        crafted;
}
function showPetalTip(def, ev) {
    if (!petalTipEl) return;
    petalTipEl.innerHTML = petalTipHTML(def);
    petalTipEl.style.display = 'block';
    movePetalTip(ev);
}
function movePetalTip(ev) {
    if (!petalTipEl || petalTipEl.style.display === 'none') return;
    const x = Math.min(ev.clientX + 14, window.innerWidth - petalTipEl.offsetWidth - 8);
    const y = Math.min(ev.clientY + 16, window.innerHeight - petalTipEl.offsetHeight - 8);
    petalTipEl.style.left = x + 'px';
    petalTipEl.style.top = y + 'px';
}
function hidePetalTip() {
    if (petalTipEl) petalTipEl.style.display = 'none';
}
// Страховка от «зависшего» тултипа: если петаль под курсором удалили/перерисовали
// (кнопка X, выброс, крафт, ререндер) — mouseleave на старом элементе не
// срабатывает, и окошко остаётся. Прячем его, как только курсор оказывается
// вне любого элемента с классом tip-anchor (петаль-слоты). Срабатывает на
// любом mouseover/mouseout, включая переход на то, что осталось под курсором.
function hideTipUnlessOnAnchor(ev) {
    if (!petalTipEl || petalTipEl.style.display === 'none') return;
    const t = ev.target;
    if (!(t instanceof Element) || !t.closest('.tip-anchor')) hidePetalTip();
}
document.addEventListener('mouseover', hideTipUnlessOnAnchor);
document.addEventListener('mouseout', hideTipUnlessOnAnchor);
function attachPetalTip(el, def) {
    el.title = '';
    el.classList.add('tip-anchor');
    el.addEventListener('mouseenter', (ev) => showPetalTip(def, ev));
    el.addEventListener('mousemove', movePetalTip);
    el.addEventListener('mouseleave', hidePetalTip);
}

// ----- Админ-панель -----
let amAdmin = false;
const adminBtn = document.getElementById('adminBtn');
const adminPanel = document.getElementById('adminPanel');
const adminAckEl = document.getElementById('adminAck');

function setUpAdmin(isAdmin) {
    amAdmin = !!isAdmin;
    if (adminBtn) adminBtn.style.display = amAdmin ? 'flex' : 'none';
}

// Отображение бейджа админа (неуязвимость/невидимость).
function setAdminState(god, invisible, passive, team, oneshot, size) {
    const g = document.getElementById('abGod');
    const v = document.getElementById('abInvis');
    const ps = document.getElementById('abPassive');
    const pv = document.getElementById('abPvp');
    const mt = document.getElementById('abMobTeam');
    const os = document.getElementById('abOneShot');
    if (g) g.style.display = god ? 'inline-block' : 'none';
    if (v) v.style.display = invisible ? 'inline-block' : 'none';
    if (ps) ps.style.display = passive ? 'inline-block' : 'none';
    if (pv) pv.style.display = (team === 1) ? 'inline-block' : 'none';
    if (mt) mt.style.display = (team === 2) ? 'inline-block' : 'none';
    if (os) os.style.display = oneshot ? 'inline-block' : 'none';
}

adminBtn && adminBtn.addEventListener('click', () => {
    adminPanel.classList.toggle('open');
    if (adminPanel.classList.contains('open')) { requestAdminPlayers(); requestAdminMobs(); startAdminMobsLive(); }
    else stopAdminMobsLive();
});

// Живой таймер Fractal в админ-панели: пока панель открыта, раз в секунду
// перезапрашиваем adminMobs, чтобы таймер шёл в реальном времени.
let admMobsTimer = null;
function startAdminMobsLive() {
    if (admMobsTimer) return;
    admMobsTimer = setInterval(requestAdminMobs, 1000);
}
function stopAdminMobsLive() {
    if (admMobsTimer) { clearInterval(admMobsTimer); admMobsTimer = null; }
}

function sendAdmin(obj) {
    if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify(obj));
    }
}

function requestAdminPlayers() {
    sendAdmin({ type: 'adminPlayers' });
}

function renderAdminPlayers(players) {
    const box = document.getElementById('admPlayers');
    if (!box) return;
    if (!players || players.length === 0) {
        box.innerHTML = '<div class="adm-row" style="color:#8a93a6">Nobody is online</div>';
        return;
    }
    box.innerHTML = '';
    const zoneN = zones.length ? zones.map(z => z.name) : ['Point','Line','Triangle','Square','Pentagon','Hexagon','Fractal','Chaos','Point II','Line II','Triangle II','Square II','Pentagon II','Hexagon II','Fractal II','Chaos II'];
    for (const p of players) {
        const row = document.createElement('div');
        row.className = 'adm-row';
        const left = document.createElement('div');
        const nm = document.createElement('div');
        nm.className = 'nm';
        nm.textContent = p.name;
        const info = document.createElement('div');
        info.className = 'info';
        info.textContent = `HP ${Math.round(p.hp)}/${Math.round(p.maxHp)} · lvl ${p.level} · ${zoneN[p.zone]||p.zone}`;
        left.appendChild(nm);
        left.appendChild(info);
        const right = document.createElement('div');
        right.style.cssText = 'display:flex;gap:6px';
        const pull = document.createElement('button');
        pull.className = 'adm-btn mini';
        pull.textContent = 'To me';
        pull.addEventListener('click', () => { sendAdmin({ type: 'adminPull', name: p.name }); });
        const ban = document.createElement('button');
        ban.className = 'adm-btn mini ban';
        ban.textContent = 'Ban';
        ban.addEventListener('click', () => {
            if (confirm(`Ban ${p.name}?`)) sendAdmin({ type: 'adminBan', name: p.name });
        });
        right.appendChild(pull);
        right.appendChild(ban);
        row.appendChild(left);
        row.appendChild(right);
        box.appendChild(row);
    }
}

document.getElementById('admSpawnBtn').addEventListener('click', () => {
    const kind = document.getElementById('admKind').value;
    const rarity = parseInt(document.getElementById('admRarity').value, 10) || 0;
    const mtype = document.getElementById('admMobType').value;
    sendAdmin({ type: 'adminSpawn', kind, rarity, mtype });
});

document.getElementById('admRefreshBtn').addEventListener('click', requestAdminPlayers);

document.getElementById('admUnbanBtn').addEventListener('click', () => {
    const n = document.getElementById('admUnbanNick').value.trim();
    if (n) { sendAdmin({ type: 'adminUnban', name: n }); document.getElementById('admUnbanNick').value = ''; }
});

document.getElementById('admSpeedBtn').addEventListener('click', () => {
    const v = parseFloat(document.getElementById('admSpeed').value);
    if (!isNaN(v)) sendAdmin({ type: 'adminSpeed', value: v });
});

document.getElementById('admHPBtn').addEventListener('click', () => {
    const v = parseFloat(document.getElementById('admHP').value);
    if (!isNaN(v)) sendAdmin({ type: 'adminHP', value: v });
});

// ---- Новые админ-контролы ----
document.getElementById('admAnnounceBtn').addEventListener('click', () => {
    const t = document.getElementById('admAnnounce').value.trim();
    if (t) { sendAdmin({ type: 'adminAnnounce', text: t }); document.getElementById('admAnnounce').value = ''; }
});

document.getElementById('admGodBtn').addEventListener('click', () => sendAdmin({ type: 'adminGod' }));
document.getElementById('admInvisibleBtn').addEventListener('click', () => sendAdmin({ type: 'adminInvisible' }));
document.getElementById('admPassiveBtn').addEventListener('click', () => sendAdmin({ type: 'adminPassive' }));
document.getElementById('admPvpBtn').addEventListener('click', () => sendAdmin({ type: 'adminPvp' }));
document.getElementById('admMobTeamBtn').addEventListener('click', () => sendAdmin({ type: 'adminMobTeam' }));
document.getElementById('admOneShotBtn').addEventListener('click', () => sendAdmin({ type: 'adminOneShot' }));
document.getElementById('admGridBtn').addEventListener('click', () => {
    const on = toggleGrid();
    saveSettings();
    if (adminAckEl) adminAckEl.textContent = 'Grid: ' + (on ? 'on' : 'off');
});

document.getElementById('admSizeInc').addEventListener('click', (e) => sendAdmin({ type: 'adminSize', delta: e.shiftKey ? 1 : 5 }));
document.getElementById('admSizeDec').addEventListener('click', (e) => sendAdmin({ type: 'adminSize', delta: e.shiftKey ? -1 : -5 }));

// Слайдер дальности видимости камеры (только для админа — отладочный инструмент).
(function () {
    const sl = document.getElementById('admViewSlider');
    const lb = document.getElementById('admViewLabel');
    if (!sl) return;
    sl.addEventListener('input', () => {
        adminViewDist = parseFloat(sl.value);
        if (lb) lb.textContent = Math.round(adminViewDist);
    });
})();

document.getElementById('admTeleportBtn').addEventListener('click', () => {
    const n = document.getElementById('admTeleportInput').value.trim();
    if (n) sendAdmin({ type: 'adminTeleport', name: n });
});

document.getElementById('admPullBtn').addEventListener('click', () => {
    const n = document.getElementById('admPullInput').value.trim();
    if (n) { sendAdmin({ type: 'adminPull', name: n }); document.getElementById('admPullInput').value = ''; }
});

// Кнопки «Телепорт в центр зоны» (Chaos/Fractal/Pentagon/Line/Point).
// Кнопки «В центр Safe Zone» (.adm-safe-btn) живут своей веткой и отправляют
// adminZoneSafe, чтобы не срабатывал двойной телепорт в центр зоны.
document.querySelectorAll('.adm-zone-btn:not(.adm-safe-btn)').forEach(btn => {
    btn.addEventListener('click', () => {
        const zone = parseInt(btn.dataset.zone, 10);
        if (!isNaN(zone)) sendAdmin({ type: 'adminZoneTel', zone });
    });
});

document.querySelectorAll('.adm-safe-btn').forEach(btn => {
    btn.addEventListener('click', () => {
        const zone = parseInt(btn.dataset.zone, 10);
        if (!isNaN(zone)) sendAdmin({ type: 'adminZoneSafe', zone });
    });
});

document.getElementById('admGivePetalsBtn').addEventListener('click', () => {
    const kind = document.getElementById('admGpKind').value;
    const rarity = parseInt(document.getElementById('admGpRarity').value, 10) || 0;
    const count = parseInt(document.getElementById('admGpCount').value, 10) || 1;
    sendAdmin({ type: 'adminGivePetals', kind, rarity, count });
});

document.getElementById('admXPBtn').addEventListener('click', () => {
    const v = parseFloat(document.getElementById('admXP').value);
    if (!isNaN(v)) sendAdmin({ type: 'adminGiveXP', value: v });
});

function requestAdminMobs() { sendAdmin({ type: 'adminMobs' }); }
document.getElementById('admMobsRefreshBtn').addEventListener('click', requestAdminMobs);

function renderAdminMobs(zones) {
    const box = document.getElementById('admMobs');
    if (!box) return;
    if (!zones || zones.length === 0) {
        box.innerHTML = '<div class="adm-row" style="color:#8a93a6">No data</div>';
        return;
    }
    box.innerHTML = '';
    for (const z of zones) {
        const row = document.createElement('div');
        row.className = 'adm-row';
        const left = document.createElement('div');
        const nm = document.createElement('div');
        nm.className = 'nm';
        nm.textContent = z.name;
        const info = document.createElement('div');
        info.className = 'info';
        info.textContent = `${z.count}/${z.limit} mobs`;
        // Статус Hexagon-мини-боссов зоны — ОТДЕЛЬНОЙ строкой (кап 2).
        const fr = document.createElement('div');
        fr.className = 'info';
        const hh = z.hexHere || 0;
        fr.textContent = '⚔ Hexagon: ' + hh + '/2 в зоне';
        fr.style.color = hh > 0 ? '#2ecc71' : '#8a93a6';
        left.appendChild(nm);
        left.appendChild(info);
        left.appendChild(fr);
        row.appendChild(left);
        box.appendChild(row);
    }
}

document.addEventListener('keydown', (e) => {
    if (!amAdmin || e.repeat) return;
    const tag = (document.activeElement && document.activeElement.tagName) || '';
    if (tag === 'INPUT' || tag === 'TEXTAREA') return;
    // N — невидимость, H — неуязвимость (админ).
    // По физическому коду клавиши — работает на любой раскладке.
    if (e.code === 'KeyN') {
        sendAdmin({ type: 'adminInvisible' });
    } else if (e.code === 'KeyH') {
        sendAdmin({ type: 'adminGod' });
    }
});

// Открытие панели по клавише B (когда нет ввода).
adminBtn && document.addEventListener('keydown', (e) => {
    if (!amAdmin || e.repeat) return;
    const tag = (document.activeElement && document.activeElement.tagName) || '';
    if (tag === 'INPUT' || tag === 'TEXTAREA') return;
    if (e.code === 'KeyB') {
        adminPanel.classList.toggle('open');
        if (adminPanel.classList.contains('open')) { requestAdminPlayers(); requestAdminMobs(); startAdminMobsLive(); }
        else stopAdminMobsLive();
    }
});

    function openChat() {
        chatOpen = true;
        chatWindow.style.display = 'flex';
        chatButton.style.display = 'none';
        // Пока открыт полный чат — всплывающие тосты полностью скрыты:
        // исчезающие сообщения живут внутри самого лога чата.
        if (chatToastEl) chatToastEl.style.display = 'none';
        // Чат всегда открывается на САМОМ КОНЦЕ (свежие сообщения внизу).
        chatLog.scrollTop = chatLog.scrollHeight;
        chatInput.blur();
        chatInput.focus();
    }

    function closeChat() {
        chatOpen = false;
        chatWindow.style.display = 'none';
        chatButton.style.display = 'flex';
        if (chatToastEl) chatToastEl.style.display = '';
        chatInput.value = '';
        if (chatInput === document.activeElement) chatInput.blur();
    }

function toggleChat() {
    if (chatOpen) closeChat();
    else openChat();
}

function sendChat() {
    const text = chatInput.value.trim();
    if (text && ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'chat', text }));
    }
    closeChat();
}

// Фокус-фикс: после клика по кнопкам панелей снимаем фокус, иначе «Space»
// нажимает кнопку снова (открыть/закрыть) — неудобно в бою.
function panelBtnClick(fn, btn) {
    return (ev) => { fn(); if (btn && btn.blur) btn.blur(); };
}
chatButton.addEventListener('click', openChat);
// Внутриигровой инвентарь и крафт: кнопки и закрытие.
const gameInvBtn = document.getElementById('gameInvBtn');
if (gameInvBtn) gameInvBtn.addEventListener('click', panelBtnClick(toggleGameInv, gameInvBtn));
const gameInvCloseBtn = document.getElementById('gameInvClose');
if (gameInvCloseBtn) gameInvCloseBtn.addEventListener('click', closeGameInv);
const gameCraftBtnEl = document.getElementById('gameCraftBtn');
if (gameCraftBtnEl) gameCraftBtnEl.addEventListener('click', panelBtnClick(toggleCraftPanel, gameCraftBtnEl));
const gameInvSearchEl = document.getElementById('gameInvSearch');
if (gameInvSearchEl) gameInvSearchEl.addEventListener('input', () => {
    window.__gameInvQ = gameInvSearchEl.value || '';
    updateGameInvPanel(true);
});
chatInput.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') {
        e.preventDefault();
        e.stopPropagation();
        sendChat();
    } else {
        e.stopPropagation();
    }
});

// ----- Экран перехода (плавное затемнение до чёрного и обратно) -----
let transferTimer = null;
let transferStart = 0;
const FADE = 500;        // длительность затемнения/растерлевания (мс)
const HOLD_BLACK = 1700; // сколько держим полный чёрный (мс)

function startTransfer(fromIdx, toIdx) {
    const fromName = zones[fromIdx] ? zones[fromIdx].name : '???';
    const toName = zones[toIdx] ? zones[toIdx].name : '???';
    transferText.textContent = `Transferring from ${fromName} to ${toName}!`;

    if (myId && actors[myId]) actors[myId].sprite.visible = false;

    transferEl.style.display = 'flex';
    transferEl.style.opacity = '0';
    transferStart = Date.now();

    if (transferTimer) cancelAnimationFrame(transferTimer);
    transferTimer = requestAnimationFrame(transferTick);
}

function transferTick() {
    const elapsed = Date.now() - transferStart;

    // Всего: FADE (в чёрный) + HOLD_BLACK (чёрный) + FADE (из чёрного).
    const total = FADE + HOLD_BLACK + FADE;
    let opacity;

    if (elapsed < total) {
        if (elapsed < FADE) {
            // Затемнение: 0 -> 1 (до полной черноты).
            opacity = elapsed / FADE;
        } else if (elapsed < FADE + HOLD_BLACK) {
            // Держим полный чёрный.
            opacity = 1;
        } else {
            // Растерлевание: 1 -> 0.
            opacity = 1 - (elapsed - FADE - HOLD_BLACK) / FADE;
        }
        transferEl.style.opacity = opacity.toFixed(3);
        transferTimer = requestAnimationFrame(transferTick);
        return;
    }

    // Конец.
    transferEl.style.display = 'none';
    if (myId && actors[myId]) actors[myId].sprite.visible = true;
}

// ----- Миникарта (круглый радар: показывает только то, что видит камера) -----
function drawMinimap() {
    const W = minimapCanvas.width;
    const H = minimapCanvas.height;
    mctx.clearRect(0, 0, W, H);

    if (myZone < 0 || !zones[myZone]) return;
    const z = zones[myZone];

    // Центр и радиус круга миникарты.
    const cx = W / 2, cy = H / 2;
    const R = Math.min(W, H) / 2 - 6;

    // Окно миникарты = прилегающая к игроку область, но НЕ шире зоны.
    // Если зона уже экрана — окно = вся зона (заполняет круг), если больше —
    // окно = камера-область вокруг игрока.
    const vW = Math.min(app.screen.width, z.w);
    const vH = Math.min(app.screen.height, z.h);

    // Центр окна: по умолчанию игрок, но клампим так, чтобы окно не вылезало
    // за границы зоны (как камера на экране у стены). Поэтому у края зоны
    // игрок смещается от центра, а за границей зоны — ничего не видно.
    let camX = 0, camY = 0;
    if (myId && actors[myId]) { camX = actors[myId].sprite.x; camY = actors[myId].sprite.y; }
    if (z.w <= vW) camX = z.x + z.w / 2;
    else camX = Math.min(Math.max(camX, z.x + vW / 2), z.x + z.w - vW / 2);
    if (z.h <= vH) camY = z.y + z.h / 2;
    else camY = Math.min(Math.max(camY, z.y + vH / 2), z.y + z.h - vH / 2);

    const scale = (R * 2) / Math.max(vW, vH);
    // Референс — ЦЕНТР окна (camX/camY): тогда игрок, когда окно не зажато
    // (camX==me.x), оказывается ровно в центре круга cx/cy.
    const sx = (x) => cx + (x - camX) * scale;
    const sy = (y) => cy + (y - camY) * scale;

    // Круглая маска + подложка.
    mctx.save();
    mctx.beginPath();
    mctx.arc(cx, cy, R, 0, Math.PI * 2);
    mctx.clip();
    mctx.fillStyle = 'rgba(8,12,18,0.92)';
    mctx.fillRect(0, 0, W, H);

    // Рамка зоны (клип по кругу сам обрежет всё лишнее).
    const zx = sx(z.x), zy = sy(z.y), zw = z.w * scale, zh = z.h * scale;
    mctx.strokeStyle = 'rgba(255,255,255,0.5)';
    mctx.lineWidth = 1;
    mctx.strokeRect(zx, zy, zw, zh);

    // Safe Zone: светлая полоса внизу зоны + отметка прохода по центру.
    if (z.safeH > 0) {
        const safeTop = z.h - z.safeH;
        const zy2 = sy(z.y + safeTop);
        mctx.strokeStyle = 'rgba(125,255,176,0.7)';
        mctx.lineWidth = 1;
        mctx.beginPath();
        mctx.moveTo(zx + 1, zy2);
        mctx.lineTo(zx + zw - 1, zy2);
        mctx.stroke();
        if (z.wallGap > 0) {
            const half = z.wallGap / 2;
            const cx = sx(z.x + z.w / 2);
            const wy2 = sy(z.y + safeTop - z.wallTh);
            mctx.strokeStyle = 'rgba(125,255,176,0.45)';
            mctx.beginPath();
            mctx.moveTo(cx - half * scale, wy2);
            mctx.lineTo(cx - half * scale, zy2);
            mctx.moveTo(cx + half * scale, wy2);
            mctx.lineTo(cx + half * scale, zy2);
            mctx.stroke();
        }
    }

    // Конфигурируемые стены зоны на миникарте.
    if (z.walls) {
        mctx.fillStyle = 'rgba(255,130,90,0.85)';
        for (const wl of z.walls) {
            const wx = sx(z.x + wl.x), wy = sy(z.y + wl.y);
            mctx.fillRect(wx, wy, wl.w * scale, wl.h * scale);
        }
    }

    // Портал(ы) этой зоны - маленькие яркие метки.
    mctx.fillStyle = '#ffd54f';
    for (const p of portals) {
        if (p.from === myZone) {
            mctx.fillRect(sx(p.x) - 2, sy(p.y) - 2, 4, 4);
        }
    }

    // Игроки (кроме меня) в видимой области.
    for (const id in actors) {
        const a = actors[id];
        if (id === myId || a.me) continue;
        const mx = sx(a.sprite.x), my2 = sy(a.sprite.y);
        if (mx < cx - R || mx > cx + R || my2 < cy - R || my2 > cy + R) continue;
        mctx.fillStyle = '#aaa';
        mctx.beginPath();
        mctx.arc(mx, my2, 2, 0, Math.PI * 2);
        mctx.fill();
    }

    // Мобы в видимой области. Дружелюбные миньоны: СВОИ — зелёные,
    // ЧУЖИЕ — синие; враждебные — красные.
    for (const id in mobActors) {
        const a = mobActors[id];
        const mx = sx(a.c.x), my2 = sy(a.c.y);
        if (mx < cx - R || mx > cx + R || my2 < cy - R || my2 > cy + R) continue;
        if (a.ally) {
            mctx.fillStyle = (a.owner && a.owner === myId) ? '#2ecc71' : '#4a9bff';
        } else {
            mctx.fillStyle = '#ff6b6b';
        }
        mctx.beginPath();
        mctx.arc(mx, my2, 1.5 + Math.min(4, a.size * 0.03), 0, Math.PI * 2);
        mctx.fill();
    }

    // Петали-предметы в видимой области.
    for (const id in groundActors) {
        const a = groundActors[id];
        const mx = sx(a.c.x), my2 = sy(a.c.y);
        if (mx < cx - R || mx > cx + R || my2 < cy - R || my2 > cy + R) continue;
        mctx.fillStyle = '#ffd700';
        mctx.beginPath();
        mctx.arc(mx, my2, 1.8, 0, Math.PI * 2);
        mctx.fill();
    }

    // Я - рисуюсь через свою реальную позицию (не жёстко в центре):
    // у края зоны камера зажата, и я могу быть не по центру миникарты.
    if (myId && actors[myId]) {
        const me = actors[myId];
        const mx = sx(me.sprite.x), my2 = sy(me.sprite.y);
        mctx.fillStyle = '#fff';
        mctx.beginPath();
        mctx.arc(mx, my2, 3.5, 0, Math.PI * 2);
        mctx.fill();
    }

    mctx.restore();

    // Круглая рамка.
    mctx.beginPath();
    mctx.arc(cx, cy, R, 0, Math.PI * 2);
    mctx.strokeStyle = 'rgba(255,255,255,0.4)';
    mctx.lineWidth = 2;
    mctx.stroke();
}

// ----- Игровой цикл -----
const keys = { w: false, a: false, s: false, d: false };

// ----- Настройки -----
let mouseControl = false; // движение к курсору (иначе WASD)
let showZone = false; // надпись "You are in ... Zone" — по умолчанию ВЫКЛЮЧЕНА
let showMinimap = true; // показывать миникарту
// Хитбоксы (для отладки/наглядности): мобы, петали боссов, петали игроков, игроки.
let showHitMobs = false;
let showHitBossPetals = false;
let showHitPlayerPetals = false;
let showHitPlayers = false;
const SETTINGS_KEY = 'shaperSettings';

// ---- Дальность видимости камеры (единая для всех) ----
const baseViewDist = 1500;    // базовая дальность (мир-единиц), видна по умолчанию
let adminViewDist = baseViewDist; // текущая дальность (слайдер админа меняет)
let currentViewScale = 1;     // масштаб контейнера мира (вычисляется каждый кадр)
let currentEffW = baseViewDist; // эффективная ширина видимости (для сервера)
let currentEffH = baseViewDist; // эффективная высота видимости (для сервера)

// Линия (петаль line, антенна на игроке): обзор растёт геометрически.
const lineViewCap = 20000; // потолок (мир-единиц) — защита от лагов на r7+
let myLineMult = 1;        // текущий множитель (плавно догоняет myLineTarget)
let myLineTarget = 1;      // целевой множитель (1.1^редкость живой line)

// Компас (петаль compass): направление (мировой угол, рад) и дистанция до
// ближайшего Fractal/Chaos-моба. обновляется сообщением "compass" с сервера.
let compassInfo = null;

// effViewDist — фактическая дальность камеры с учётом антенны-line (множитель
// перекрывает и админ-слайдер), зажата потолком lineViewCap.
function effViewDist() {
    const v = adminViewDist * (myLineMult || 1);
    return Math.min(v, lineViewCap);
}

function loadSettings() {
    let s = {};
    try { s = JSON.parse(localStorage.getItem(SETTINGS_KEY) || '{}'); } catch (e) { s = {}; }
    mouseControl = !!s.mouse;
    gridOn = !!s.grid;
    if (typeof s.zone === 'boolean') showZone = s.zone;
    if (typeof s.minimap === 'boolean') showMinimap = s.minimap;
    if (typeof s.hitMobs === 'boolean') showHitMobs = s.hitMobs;
    if (typeof s.hitBossPetals === 'boolean') showHitBossPetals = s.hitBossPetals;
    if (typeof s.hitPlayerPetals === 'boolean') showHitPlayerPetals = s.hitPlayerPetals;
    if (typeof s.hitPlayers === 'boolean') showHitPlayers = s.hitPlayers;
}
function saveSettings() {
    localStorage.setItem(SETTINGS_KEY, JSON.stringify({
        mouse: mouseControl, grid: gridOn, zone: showZone, minimap: showMinimap,
        hitMobs: showHitMobs, hitBossPetals: showHitBossPetals,
        hitPlayerPetals: showHitPlayerPetals, hitPlayers: showHitPlayers,
    }));
}
function applyZoneLabel() {
    zoneLabel.style.display = showZone ? '' : 'none';
    // Босс-бары: если надпись зоны видна — они чуть НИЖЕ неё, чтобы не наезжать.
    const bh = document.getElementById('bossHud');
    if (bh) bh.classList.toggle('below-label', showZone);
}
function applyMinimap() {
    minimapCanvas.style.display = showMinimap ? '' : 'none';
}
function syncSettingsUI() {
    const sg = document.getElementById('setGrid');
    const sm = document.getElementById('setMouse');
    const sz = document.getElementById('setZone');
    const smm = document.getElementById('setMinimap');
    const hm = document.getElementById('setHitMobs');
    const hbp = document.getElementById('setHitBossPetals');
    const hpp = document.getElementById('setHitPlayerPetals');
    const hp = document.getElementById('setHitPlayers');
    if (sg) sg.checked = gridOn;
    if (sm) sm.checked = mouseControl;
    if (sz) sz.checked = showZone;
    if (smm) smm.checked = showMinimap;
    if (hm) hm.checked = showHitMobs;
    if (hbp) hbp.checked = showHitBossPetals;
    if (hpp) hpp.checked = showHitPlayerPetals;
    if (hp) hp.checked = showHitPlayers;
}

document.getElementById('settingsBtn').addEventListener('click', () => {
    const p = document.getElementById('settingsPanel');
    syncSettingsUI();
    p.classList.add('open');
});
document.getElementById('settingsCloseBtn').addEventListener('click', () => {
    document.getElementById('settingsPanel').classList.remove('open');
    saveSettings();
    applyZoneLabel();
    applyMinimap();
});
document.getElementById('setGrid').addEventListener('change', (e) => {
    gridOn = e.target.checked;
    rebuildGrid();
    saveSettings();
});
document.getElementById('setMouse').addEventListener('change', (e) => {
    mouseControl = e.target.checked;
    saveSettings();
});
document.getElementById('setZone').addEventListener('change', (e) => {
    showZone = e.target.checked;
    saveSettings();
    applyZoneLabel();
});
document.getElementById('setMinimap').addEventListener('change', (e) => {
    showMinimap = e.target.checked;
    saveSettings();
    applyMinimap();
});
document.getElementById('setHitMobs').addEventListener('change', (e) => {
    showHitMobs = e.target.checked;
    saveSettings();
});
document.getElementById('setHitBossPetals').addEventListener('change', (e) => {
    showHitBossPetals = e.target.checked;
    saveSettings();
});
document.getElementById('setHitPlayerPetals').addEventListener('change', (e) => {
    showHitPlayerPetals = e.target.checked;
    saveSettings();
});
document.getElementById('setHitPlayers').addEventListener('change', (e) => {
    showHitPlayers = e.target.checked;
    saveSettings();
});
loadSettings();
applyZoneLabel();
applyMinimap();

// Позиция мыши в мировых координатах (для управления мышью).
let mouseWorldX = 0, mouseWorldY = 0;
app.view.addEventListener('pointermove', (e) => {
    const rect = app.view.getBoundingClientRect();
    const sx = (e.clientX - rect.left) / rect.width * app.screen.width;
    const sy = (e.clientY - rect.top) / rect.height * app.screen.height;
    mouseWorldX = (sx - world.x) / currentViewScale;
    mouseWorldY = (sy - world.y) / currentViewScale;
});

function tick() {
    const lerp = 0.25;

    // Плавное (60 раз/с) изменение обзора от антенны-line: множитель
    // экспоненциально догоняет целевой — FOV растёт/уменьшается без рывков.
    myLineMult += (myLineTarget - myLineMult) * 0.035;
    if (Math.abs(myLineTarget - myLineMult) < 0.001) myLineMult = myLineTarget;

    // Плавное растворение сущностей, пропавших из state.
    tickFadeOuts(app.ticker.deltaMS);

    // Кулдаун петалей в хотбаре: обновляем «вуаль» сверху вниз по доле перезарядки.
    for (const slot in gameHotbarVeils) {
        const v = gameHotbarVeils[slot];
        if (!v) continue;
        const pct = (hotbarSlotCD[slot] || 0) * 100;
        if (v.pct !== pct) {
            v.pct = pct;
            v.style.height = pct + '%';
        }
    }

    for (const id in actors) {
        const a = actors[id];
        a.sprite.x += (a.targetX - a.sprite.x) * lerp;
        a.sprite.y += (a.targetY - a.sprite.y) * lerp;
    }
    // Пузыри чата: синхронизируем ПОСЛЕ лерпа спрайтов — позиция совпадает
    // с игроком пиксель в пиксель каждый кадр, без запаздывания и рывков.
    // Слои: мобы/дропы → ИГРОКИ + их петали + снаряды + цепи (переподнимаем
    // наверх — петали часть игрока и не должны прятаться под мобами) → пузыри.
    for (const id in actors) {
        const a = actors[id];
        if (a.sprite.parent === world) world.addChild(a.sprite);
    }
    for (const id in petalActors) {
        const a = petalActors[id];
        if (a.g.parent === world) world.addChild(a.g);
    }
    for (const id in projectileActors) {
        const a = projectileActors[id];
        if (a.c.parent === world) world.addChild(a.c);
    }
    if (chainGfx && chainGfx.parent === world) world.addChild(chainGfx);
    ensureBubbleLayer();
    if (chatBubbleLayer && chatBubbleLayer.parent) {
        for (const b of chatBubbleLayer.children) {
            const a = actors[b.actorId];
            if (a) b.position.set(a.sprite.x, a.sprite.y + b.offY);
        }
    }
    for (const id in mobActors) {
        const a = mobActors[id];
        a.c.x += (a.targetX - a.c.x) * lerp;
        a.c.y += (a.targetY - a.c.y) * lerp;

        // Спец-аура Fractal/Chaos — пульс и орбитальные петали (боссов) каждый кадр.
        // Углы петалей интерполируются между приходами стейтов (сервер шлёт ~12 Гц),
        // чтобы вращение было плавным, без рывков. Каждый петаль идентифицируется
        // СВОИМ стабильным idx (не позицией в массиве): когда петаль умирает и не
        // приходит в состоянии, а потом возрождается, остальные НЕ смещаются — иначе
        // из-за смены индексов визуально петали «прыгали» на соседние места орбиты.
        if (a.aura) {
            const bp = a.bossPetals || [];
            if (!a.petalCur) a.petalCur = {};
            for (const p of bp) {
                const key = 'p' + p.idx;
                const cur = a.petalCur[key];
                if (cur == null) {
                    a.petalCur[key] = p.angle;
                } else {
                    const d = Math.atan2(Math.sin(p.angle - cur), Math.cos(p.angle - cur));
                    a.petalCur[key] = cur + d * 0.35;
                }
            }
            // Орбитальные петали боссов: у каждого СВОЙ график, форма повторяет тип
            // моба (Square Fractal -> квадратная петаль, Entropy Chaos -> «энтропия»
            // и т.д.), цвет — цвет типа моба. Аура-очерк редкости: Fractal фиолет,
            // Chaos малиновый. Петаль гаснет по мере урона (hp -> близко к 0).
            if (!a.petalSpr) a.petalSpr = {};
            const rr = a.size * 1.9;
            const rarCol = a.rarityI >= 7 ? 0xff2d95 : 0x9b59b6;
            const psz = Math.max(4, a.size * 0.3);
            for (const p of bp) {
                const key = 'p' + p.idx;
                const ang = a.petalCur[key] != null ? a.petalCur[key] : p.angle;
                let spr = a.petalSpr[key];
                if (!spr) {
                    spr = new PIXI.Graphics();
                    a.aura.addChild(spr);
                    a.petalSpr[key] = spr;
                }
                spr.x = Math.cos(ang) * rr;
                spr.y = Math.sin(ang) * rr;
                spr.rotation = ang;
                drawMobShape(spr, { shape: a.shape || 'circle', size: psz, color: a.color != null ? a.color : 0x888888 });
                // Урон НЕ делаем прозрачностью: петаль всегда плотная, урон читается
                // по цвету окантовки (гаснет с HP) — см. lineStyle ниже.
                spr.alpha = 1;
                const hpr = p.hp / Math.max(1, p.maxHp);
                spr.lineStyle(2, rarCol, 0.2 + 0.4 * hpr);
                spr.drawCircle(0, 0, psz * 0.95);
                spr.lineStyle(0);
                // Хитбокс петали босса: круг реального радиуса сервера (p.size).
                if (showHitBossPetals) {
                    const hitR = (p.size != null) ? p.size : psz;
                    spr.lineStyle(1.5, 0xff2d95, 0.8);
                    spr.drawCircle(0, 0, hitR);
                    spr.lineStyle(0);
                }
                if (p.poison) {
                    spr.lineStyle(1.5, 0x39ff5e, 0.9);
                    spr.drawCircle(0, 0, psz + 7);
                }
            }
            // Уничтоженные петали сервер больше не шлёт — плавно растворяем их
            // (как мобов и петали), при возрождении на том же idx график создастся заново.
            for (const id of Object.keys(a.petalSpr)) {
                if (!bp.some(p => 'p' + p.idx === id)) {
                    fadeOutThenRemove(a.petalSpr[id]);
                    delete a.petalSpr[id];
                }
            }
            drawMobAura(a.aura, { size: a.size, rarityI: a.rarityI }, performance.now() / 1000);
        }

        // Витринные Comet (14) и Beam (15) в TestRoom: «агро» — при приближении
        // игрока поворачиваются ГОЛОВОЙ на него (хвост/излучатель остаются сзади).
        // Сервер этих мокапов не двигает (Speed=0), направление считаем сами.
        if (a.showcase && (a.label === '14' || a.label === '15')) {
            let face = null, faceD2 = Number.MAX_VALUE;
            for (const pid in actors) {
                const pa = actors[pid];
                const ddx = pa.targetX - a.c.x, ddy = pa.targetY - a.c.y;
                const d2 = ddx * ddx + ddy * ddy;
                if (d2 < faceD2) { faceD2 = d2; face = Math.atan2(ddy, ddx); }
            }
            if (face != null && faceD2 < 1000 * 1000) a.rot = face;
        }

        // Плавный поворот к направлению движения (моб «смотрит» куда идёт).
        // FragWall (стена) поворачивается заметно медленнее остальных.
        let diff = a.rot - a.g.rotation;
        while (diff > Math.PI) diff -= Math.PI * 2;
        while (diff < -Math.PI) diff += Math.PI * 2;
        a.g.rotation += diff * (a.slowTurn ? 0.05 : 0.18);
        // Шип (spike): шипы крутятся непрерывно, а не статичны.
        if (a.shape === 'mockup_1') {
            a.g.rotation = (performance.now() / 1000 * 1.5) % (Math.PI * 2);
        }
        // Prism (mockup_16) НЕ принудительно вращается: остриё ромба смотрит
        // по направлению движения через generic-поворот (a.rot по dirX/dirY).
        // Ромбовые витринные мокапы: Jade покачивается как подвеска, Kite слегка
        // рыскает (змей на ветру). Детали дизайнов анимируются в drawShowcaseDesign.
        if (a.ghost > a.fillW) {
            a.ghost += (a.fillW - a.ghost) * 0.08;
            if (a.ghost - a.fillW < 0.8) a.ghost = a.fillW;
        }
        const gw = a.ghost - a.fillW;
        a.barGhost.clear();
        if (gw > 0.5) {
            a.barGhost.beginFill(0xfff0b0, 0.9);
            a.barGhost.drawRoundedRect(a.fillW, 0, Math.max(1.5, gw), a.bh, Math.min(a.bh / 2, gw / 2));
            a.barGhost.endFill();
        }
    }
    // Затухание убитых мобов: растворяем (альфа + уменьшаем) ~0.8 c, затем убираем.
    for (const [id, f] of mobFades) {
        f.t += 0.016;
        const p = Math.min(1, f.t / 0.8);
        f.c.alpha = Math.max(0, 1 - p);
        const s = 1 + p * 1.4;
        f.c.scale.set(s, s);
        if (p >= 1) {
            world.removeChild(f.c);
            mobFades.delete(id);
        }
    }
    for (const id in petalActors) {
        const a = petalActors[id];        a.g.x += (a.targetX - a.g.x) * lerp;
        a.g.y += (a.targetY - a.g.y) * lerp;
        // Щит (shield): петаль тает, впитываясь в игрока (пополняет полоску щита).
        if (a.shieldFade != null) a.g.alpha = a.shieldFade;
        // Кольцо «эффекта удара»: мигает, когда петаль только что атаковала.
        if (a.hitRing) {
            if (a.hit > 0) {
                a.hit -= 0.016;
                a.hitRing.clear();
                if (a.hit > 0) {
                    const p = 1 - a.hit;
                    a.hitRing.lineStyle(2, 0xffe27a, 0.8 - 0.4 * p);
                    a.hitRing.drawCircle(0, 0, (a.size || 8) + p * 16);
                    a.hitRing.lineStyle(0);
                }
            }
        }
    }
    for (const id in groundActors) {
        const a = groundActors[id];
        a.c.x += (a.targetX - a.c.x) * lerp;
        a.c.y += (a.targetY - a.c.y) * lerp;
        // Лёгкое покачивание (лежит на земле, «дышит»).
        a.bob += 0.03;
        a.c.scale.set(0.9 + 0.1 * Math.sin(a.bob));
        // Пульсация ореола у дропа редкости Fractal (фиолет) / Hexagon (янтарь)
        // + мерцающие блики-звёздочки: 9 штук мигают с разной фазой — «сверкание».
        if (a.glow) {
            const isHex5 = a.glowRar === 5;
            const glowC1 = isHex5 ? 0xffc93c : 0x9b59b6; // основной свет
            const glowC2 = isHex5 ? 0xffe08a : 0xb585d1; // контур
            const spCols = isHex5 ? [0xffd75e, 0xfff3c4, 0xffffff] : [0xb585d1, 0xd9b3ff, 0xffffff];
            const pp = 0.5 + 0.5 * Math.sin(a.bob * 1.7);
            a.glow.clear();
            a.glow.beginFill(glowC1, 0.06 + 0.08 * pp);
            a.glow.drawCircle(0, 0, a.size * 2.4);
            a.glow.endFill();
            a.glow.beginFill(glowC1, 0.09 + 0.08 * pp);
            a.glow.drawCircle(0, 0, a.size * 1.7);
            a.glow.endFill();
            a.glow.lineStyle(1.5, glowC2, 0.25 + 0.40 * pp);
            a.glow.drawCircle(0, 0, a.size * 1.25);
            a.glow.lineStyle(0);
            const cols = spCols;
            for (const s of a.sp) {
                const tw = Math.sin(a.bob * s.sp + s.ph);
                const k = tw * tw * tw * tw; // 0..1: резкие вспышки-мигания
                const sx = Math.cos(s.ang) * s.r;
                const sy = Math.sin(s.ang) * s.r;
                const sr = (0.8 + 1.7 * k) * a.size * 0.24;
                const rot = a.bob * 0.4 + s.ang;
                // Звёздочка-блик явным полигоном (drawStar есть не во всех
                // сборках PIXI, поэтому рисуем 4-конечную звезду сам).
                a.glow.beginFill(cols[s.w ? 0 : 1], 0.15 + 0.85 * k);
                a.glow.moveTo(sx + Math.cos(rot) * sr, sy + Math.sin(rot) * sr);
                for (let piV = 1; piV < 8; piV++) {
                    const rr = piV % 2 === 0 ? sr : sr * 0.45;
                    const aa = rot + (piV * Math.PI) / 4;
                    a.glow.lineTo(sx + Math.cos(aa) * rr, sy + Math.sin(aa) * rr);
                }
                a.glow.closePath();
                a.glow.endFill();
            }
        }
    }
    // Летящие снаряды (миссил-blade, осколки-hiveling): позиция из состояния,
    // ориентация по вектору скорости (angle от сервера).
    for (const id in projectileActors) {
        const a = projectileActors[id];
        a.c.x += (a.targetX - a.c.x) * lerp;
        a.c.y += (a.targetY - a.c.y) * lerp;
        a.c.rotation = a.rot;
    }
    // Плавное исчезновение подобранных петалей-дропов: взлетают, слегка
    // растворяются и тают за ~0.35 c (вместо мгновенного пропадания).
    for (const [id, f] of groundFades) {
        f.t += 0.016;
        const p = Math.min(1, f.t / 0.35);
        f.c.y -= 0.75;
        f.c.alpha = Math.max(0, 1 - p * p);
        f.c.scale.set(1 + p * 0.5, 1 + p * 0.5);
        if (p >= 1) {
            world.removeChild(f.c);
            groundFades.delete(id);
        }
    }

    // Анимация порталов: вращение спирали воронки, переливы, мерцание частиц.
    for (const h of portalContainers) {
        const t = h.userData;
        t.phase += 0.016;

        // Воронка медленно вращается (спираль + переливающиеся дуги).
        t.spiral.rotation += 0.02;
        for (const f of t.flames) f.rotation += 0.012;

        // Обод и свечение пульсируют ("дышат").
        const pulse = 1 + 0.06 * Math.sin(t.phase * 1.6);
        t.ring.scale.set(pulse, pulse);
        t.glow.scale.set(pulse * 1.05, pulse * 1.05);

        // Переливы поверхности: тон цвета меняется во времени.
        t.surface.tint = t.col;
        t.surface.alpha = 0.32 + 0.10 * Math.sin(t.phase * 2.4);

        // Мерцающие частицы летают по орбитам.
        for (const pr of t.particles) {
            pr.angle += pr.speed * 0.02 * pr.dir;
            pr.sprite.x = Math.cos(pr.angle) * pr.radius;
            pr.sprite.y = Math.sin(pr.angle) * pr.radius;
            pr.sprite.alpha = 0.5 + 0.5 * Math.sin(t.phase * 3 + pr.angle * 4);
        }
    }

    // Отправляем WASD серверу (пропускаем, когда чат открыт).
    if (ws && ws.readyState === WebSocket.OPEN && !chatOpen) {
        let ix, iy;
        if (mouseControl && myId && actors[myId]) {
            // Движение к курсору: направление от АВТОРИТЕТНОЙ позиции (target),
            // а не от интерполированного спрайта — иначе у цели игрок осциллирует.
            // Мёртвая зона = путь за один тик (speed 200 × dt 50мс = 10), чтобы
            // не дёргаться вокруг курсора, особенно у краёв зоны (камера клампится).
            const me = actors[myId];
            const ax = (me.targetX != null) ? me.targetX : me.sprite.x;
            const ay = (me.targetY != null) ? me.targetY : me.sprite.y;
            let dx = mouseWorldX - ax;
            let dy = mouseWorldY - ay;
            const len = Math.hypot(dx, dy);
            if (len > 14) { dx /= len; dy /= len; } else { dx = 0; dy = 0; }
            ix = dx; iy = dy;
        } else {
            ix = (keys.d ? 1 : 0) - (keys.a ? 1 : 0);
            iy = (keys.s ? 1 : 0) - (keys.w ? 1 : 0);
        }
        // Спам-троттлинг: серверу достаточно знать направление; пустые (0,0)
        // в простое льём не каждым кадром, а не чаще раза в 250мс (было ~60/с).
        const nw = effViewDist(), nh = effViewDist();
        const now = performance.now();
        const changed = !lastInput || lastInput.x !== ix || lastInput.y !== iy ||
                        Math.abs(lastInput.w - nw) > 1 || Math.abs(lastInput.h - nh) > 1;
        if (changed || now - lastInputT > 250) {
            lastInput = { x: ix, y: iy, w: nw, h: nh };
            lastInputT = now;
            ws.send(JSON.stringify({ type: 'input', x: ix, y: iy, viewW: nw, viewH: nh }));
        }
    }

    // Камера: дальность = effViewDist (база × антенна-line) + зум через world.scale.
    if (myId && actors[myId]) {
        const me = actors[myId];
        const sw = app.screen.width, sh = app.screen.height;
        const maxDim = Math.max(sw, sh);
        const vd = effViewDist();

        // Масштаб контейнера мира:远近 = vd по МАКСАЛЬНОМУ измерению экрана,
        // чтобы и горизонт, и вертикаль показывали одинаковое кол-во мир-единиц.
        currentViewScale = maxDim / vd;
        currentEffW = sw / currentViewScale;
        currentEffH = sh / currentViewScale;

        // Обновляем максимум слайдера под текущую зону.
        if (myZone >= 0 && zones[myZone]) {
            const sl = document.getElementById('admViewSlider');
            if (sl) {
                const mz = Math.max(zones[myZone].w, zones[myZone].h);
                if (parseFloat(sl.max) !== mz) sl.max = mz;
            }
        }

        // Центр камеры — игрок, но клампим к границам зоны.
        let camX = me.sprite.x;
        let camY = me.sprite.y;
        if (myZone >= 0 && zones[myZone]) {
            const z = zones[myZone];
            if (currentEffW < z.w) {
                camX = Math.min(Math.max(camX, z.x + currentEffW / 2), z.x + z.w - currentEffW / 2);
            } else {
                camX = z.x + z.w / 2;
            }
            if (currentEffH < z.h) {
                camY = Math.min(Math.max(camY, z.y + currentEffH / 2), z.y + z.h - currentEffH / 2);
            } else {
                camY = z.y + z.h / 2;
            }
        }

        world.scale.set(currentViewScale);
        world.x = -(camX * currentViewScale - sw / 2);
        world.y = -(camY * currentViewScale - sh / 2);
    }

    drawMinimap();
}

// ----- Меню и управление -----
// Кнопка Lobby в меню — открывает полноэкранный оверлей лобби (не входит в игру).
playBtn.addEventListener('click', () => {
    if (!authToken) { showScreen('login'); return; }
    openLobbyOverlay();
});

// enterArena — вход в круглую арену (кнопка PLAY в лобби после старта матча).
function enterArena() {
    if (!authToken) { showScreen('login'); return; }
    // Волновой режим: без лобби играть нельзя.
    if (!getLobbyCode()) {
        errorBox.textContent = 'Create or join a lobby first';
        return;
    }
    errorBox.textContent = '';
    // Плавный переход на чёрный "Loading...", затем вход в мир.
    showLoading('Loading...');
    hideDeath();
    menu.classList.remove('active');
    closeLobbyOverlay();
    syncTicker();
    pendingMenuExit = false;
    intentionalClose = false;
    if (connectTimer) { clearTimeout(connectTimer); connectTimer = null; }
    // Если вдруг осталось живое соединение — закрываем и переподключаемся.
    if (ws) { try { ws.close(); } catch (e) {} ws = null; }
    connect();
}

// Кнопка инвентаря в меню — открыть/закрыть панель инвентаря справа снизу.
document.getElementById('invBtn').addEventListener('click', toggleInvPanel);

// Enter не должен открывать чат, пока активен экран авторизации.
function authErrorIsActive() {
    return loginScreen.classList.contains('active') ||
           registerScreen.classList.contains('active') ||
           menu.classList.contains('active');
}

window.addEventListener('keydown', (e) => {
    const k = e.key ? e.key.toLowerCase() : '';
    // Не обрабатываем горячие клавиши, если пользователь печатает в поле ввода
    // (никнейм/пароль на экранах авторизации, поиск и т.п.) или открыт чат.
    const tag = (e.target && e.target.tagName) || '';
    const isTyping = tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' ||
        (e.target && e.target.isContentEditable);
    const move = { KeyW: 'w', KeyA: 'a', KeyS: 's', KeyD: 'd',
        ArrowUp: 'w', ArrowDown: 's', ArrowLeft: 'a', ArrowRight: 'd' }[e.code];
    if (move && move in keys && !isTyping) {
        keys[move] = true;
        if (e.code.startsWith('Arrow')) e.preventDefault(); // не скроллить страницу
    }
    // Enter — открыть/закрыть чат. Но не во время авторизации.
    if (e.key === 'Enter' && !authErrorIsActive() && !isTyping) {
        if (!chatOpen) {
            // Пустой Enter в поле ввода чата не должен открываться дважды.
            openChat();
            e.preventDefault();
        }
    } else if (!isTyping) {
        // Обрабатываем движение только если чат закрыт.
        if (chatOpen) return;
        if (k === 'shift') {
            if (ws && ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify({ type: 'shiftDown' }));
        }
        if (e.code === 'Space' && !e.repeat) {
            if (ws && ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify({ type: 'spaceDown' }));
        }
        // X — переключение хотбара.
        if (e.code === 'KeyX' && !e.repeat) {
            const newBar = myActiveBar === 0 ? 1 : 0;
            myActiveBar = newBar;
            renderMenuHotbar();
            renderGameHotbar();
            setHotbar(newBar);
        }
        // I — инвентарь (shapes), C — крафт (как во флор).
        if (e.code === 'KeyI' && !e.repeat) {
            toggleGameInv();
            e.preventDefault();
        }
        if (e.code === 'KeyC' && !e.repeat) {
            toggleCraftPanel();
            e.preventDefault();
        }
        // G — переключение сетки.
        if (e.code === 'KeyG' && !e.repeat) {
            toggleGrid();
            saveSettings();
            syncSettingsUI();
        }
    }
});
window.addEventListener('keyup', (e) => {
    const k = e.key ? e.key.toLowerCase() : '';
    const move = { KeyW: 'w', KeyA: 'a', KeyS: 's', KeyD: 'd',
        ArrowUp: 'w', ArrowDown: 's', ArrowLeft: 'a', ArrowRight: 'd' }[e.code];
    if (move && move in keys) keys[move] = false;
    if (chatOpen) return;
    if (k === 'shift') {
        if (ws && ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify({ type: 'shiftUp' }));
    }
    if (e.code === 'Space' && !e.repeat) {
        if (ws && ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify({ type: 'spaceUp' }));
    }
});

document.getElementById('respawnBtn').addEventListener('click', () => {
    if (ws && ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify({ type: 'respawn' }));
    hideDeath();
});

// Кнопка выхода в меню (крестик в верхнем правом углу).
const exitToMenuBtn = document.getElementById('exitToMenuBtn');
if (exitToMenuBtn) exitToMenuBtn.addEventListener('click', () => {
    hidePetalTip();
    leaveToMenu();
});

document.getElementById('menuBtn').addEventListener('click', () => {
    // Полный выход из мира: плавно затемняем, отвязываем игрока (menu→ReturnToMenu),
    // закрываем WS и переходим в меню (правки там идут через REST в БД).
    leaveToMenu();
});

// ----- Выход из мира в главное меню -----
let pendingMenuExit = false;
function leaveToMenu() {
    if (pendingMenuExit) return;
    pendingMenuExit = true;
    showLoading('Loading...');
    hideDeath();
    intentionalClose = true; // закрываем намеренно — не переподключаться
    if (connectTimer) { clearTimeout(connectTimer); connectTimer = null; }
    const hadConn = ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING);
    if (hadConn) {
        try { ws.send(JSON.stringify({ type: 'menu' })); } catch (e) {}
    } else {
        enterMenuFromWorld();
    }
    // Если соединение молчит — всё равно уходим через короткий таймаут.
    setTimeout(() => { if (pendingMenuExit) enterMenuFromWorld(); }, 1500);
}

function enterMenuFromWorld() {
    if (!pendingMenuExit) return;
    pendingMenuExit = false;
    intentionalClose = true;
    gameStarted = false;
    // Если игрок сам вышел в меню при идущем матче — не автозаходить обратно
    // из экрана пати (самовосстановление в lobby.js). Флаг снимется, когда
    // матч завершится (started=false) или хост запустит новый.
    if (typeof lobbyMatchStarted !== 'undefined' && lobbyMatchStarted &&
        typeof leaveDuringMatch !== 'undefined') {
        leaveDuringMatch = true;
    }
    myId = null;
    if (connectTimer) { clearTimeout(connectTimer); connectTimer = null; }
    if (ws) { try { ws.close(); } catch (e) {} }
    ws = null;
    const exBtn = document.getElementById('exitToMenuBtn');
    if (exBtn) exBtn.style.display = 'none';
    showScreen('menu');
    loadProfile().then(() => hideLoading());
}

window.addEventListener('resize', () => app.renderer.resize(window.innerWidth, window.innerHeight));

app.ticker.add(tick);

// Инициализация экрана при загрузке — здесь, ПОСЛЕ всех объявлений (ws и пр.).
showScreen(authToken ? 'menu' : 'login');
updatePlayerAvatar();
if (authToken) {
    loadProfile();
    startOnlinePolling();
}



// ============================================================
// SecretProjectWave: team-профили слева + Wave HUD (бар/ростер).
// Данные приходят в каждом state (20 тиков/с).
// ============================================================

let teamPanelState = ''; // кэш последнего рендера (чтобы не дёргать DOM)

function updateTeamPanel(players) {
    const panel = document.getElementById('teamPanel');
    if (!panel) return;
    const mates = (players || []).filter(p => p.id !== myId);
    // Сериализуем состояние — DOM трогаем только при изменениях.
    let key = '';
    for (const p of mates) key += p.id + '|' + p.name + '|' + p.color + '|' + Math.ceil(p.hp) + '/' + Math.ceil(p.maxHp) + '|' + (p.dead ? 'd' : 'a') + ';';
    if (key === teamPanelState) return;
    teamPanelState = key;

    panel.innerHTML = '';
    panel.style.display = mates.length ? 'flex' : 'none';
    for (const p of mates) {
        const card = document.createElement('div');
        card.className = 'tp-card' + (p.dead ? ' tp-dead' : '');
        const col = '#' + (p.color != null ? p.color.toString(16).padStart(6, '0') : '54c468');
        const hpPct = p.maxHp > 0 ? Math.max(0, Math.min(100, p.hp / p.maxHp * 100)) : 0;
        card.innerHTML =
            '<div class="tp-avatar" style="background:' + col + '"></div>' +
            '<div class="tp-main">' +
                '<div class="tp-nick">' + escapeHtml2(p.name) + '</div>' +
                '<div class="tp-hpwrap"><div class="tp-hp" style="width:' + hpPct + '%"></div></div>' +
            '</div>';
        panel.appendChild(card);
    }
}

function escapeHtml2(s) {
    return String(s || '').replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
}

let waveHudCache = '';

function updateWaveHud(state) {
    const top = document.getElementById('waveHudTop');
    if (!top) return;
    const phase = state.wavePhase || '';
    const f1 = state.waveF1 || 0;
    const f2 = state.waveF2 || 0;
    const num = state.waveNum || 0;
    const roster = state.waveRoster || [];

    const show = waveModeClient && (phase !== '' || num > 0);
    top.style.display = show ? 'block' : 'none';
    if (!show) return;

    // Заголовок.
    const star = phase === 'hold' ? ' <span class="wave-star">★</span>' : '';
    const title = 'Wave ' + num + star;
    const titleEl = document.getElementById('waveHudTitle');
    if (titleEl.textContent !== title) titleEl.innerHTML = title;

    // Полосы.
    const l1 = document.getElementById('waveBarLayer1');
    const l2 = document.getElementById('waveBarLayer2');
    l1.style.width = Math.round(f1 * 100) + '%';
    l2.style.width = Math.round(f2 * 100) + '%';
    l2.style.display = phase === 'hold' ? 'block' : 'none';

    // Ростер живых мобов: сортировка по редкости ↓, потом по названию.
    const key = roster.map(r => r.type + r.rarity + 'x' + r.count).join(',');
    if (key === waveHudCache) return;
    waveHudCache = key;
    const rosterEl = document.getElementById('waveRoster');
    rosterEl.innerHTML = '';
    const sorted = roster.slice().sort((a, b) => (b.rarity - a.rarity) || a.name.localeCompare(b.name));
    for (const r of sorted) {
        const rc = rarityColors[Math.min(r.rarity || 0, rarityColors.length - 1)];
        const chip = document.createElement('div');
        chip.className = 'wr-chip';
        chip.style.borderColor = rc;
        chip.innerHTML = '<b style="color:' + rc + '">' + escapeHtml2(r.name) + '</b>' +
            '<span class="wr-cnt">× ' + r.count + '</span>';
        rosterEl.appendChild(chip);
    }
}
