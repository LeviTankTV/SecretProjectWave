// ============================================================
// SecretProjectWave: лобби (полноэкранный оверлей) и волновой режим.
// Подключается после main.js (использует authToken, apiCall, PIXI,
// drawPetalShape, drawMobShape, rarityColors, chosenColor, enterArena).
// ============================================================

// Код текущего лобби (сохраняется между перезагрузками страницы).
function getLobbyCode() {
    return localStorage.getItem('lobbyCode') || '';
}
function setLobbyCode(code) {
    if (code) localStorage.setItem('lobbyCode', code);
    else localStorage.removeItem('lobbyCode');
}

let lobbyPollTimer = null;
let waveBannerTimer = null;
let startCountdownTimer = null;
let lobbyBusy = false;          // защита от двойных кликов по API лобби
let lobbyMatchStarted = false;
let lobbyZone = 0;              // выбранная зона (0-7)
let lobbyIsHost = false;
let lobbyWasInParty = false;    // были в пати на прошлом опросе (для «пати распалось»)
let leaveDuringMatch = false;   // сам вышел из арены при идущем матче (не автозаходить обратно)

// Зоны арены (зеркало zoneNames/zoneColors сервера).
const arenaZones = [
    { name: 'Point',   color: 0x2ecc71, css: '#2ecc71' },
    { name: 'Line',    color: 0xf1c40f, css: '#f1c40f' },
    { name: 'Triangle',color: 0xe67e22, css: '#e67e22' },
    { name: 'Square',  color: 0xe74c3c, css: '#e74c3c' },
    { name: 'Pentagon',color: 0x3498db, css: '#3498db' },
    { name: 'Hexagon', color: 0x1abc9c, css: '#1abc9c' },
    { name: 'Fractal', color: 0x9b59b6, css: '#9b59b6' },
    { name: 'Chaos',   color: 0x2c3e50, css: '#2c3e50' },
];

// Тематический фон оверлея под зону (базовый — Point).
function zoneBackgroundCss(zone) {
    switch (zone) {
        case 0: return 'radial-gradient(circle at 50% 38%, rgba(46,204,113,0.28), rgba(10,20,14,0.99) 62%), #0a140e';
        case 1: return 'radial-gradient(circle at 50% 38%, rgba(241,196,15,0.26), rgba(22,19,8,0.99) 62%), #161308';
        case 2: return 'radial-gradient(circle at 50% 38%, rgba(230,126,34,0.26), rgba(24,14,6,0.99) 62%), #180e06';
        case 3: return 'radial-gradient(circle at 50% 38%, rgba(231,76,60,0.26), rgba(26,8,6,0.99) 62%), #1a0806';
        case 4: return 'radial-gradient(circle at 50% 38%, rgba(52,152,219,0.26), rgba(6,14,24,0.99) 62%), #060e18';
        case 5: return 'radial-gradient(circle at 50% 38%, rgba(26,188,156,0.26), rgba(6,20,17,0.99) 62%), #061411';
        case 6: return 'radial-gradient(circle at 50% 38%, rgba(155,89,182,0.28), rgba(18,8,24,0.99) 62%), #120818';
        case 7: return 'radial-gradient(circle at 50% 38%, rgba(44,62,80,0.4), rgba(4,6,9,0.99) 66%), #040609';
        default: return 'radial-gradient(circle at 50% 38%, rgba(46,204,113,0.28), rgba(10,20,14,0.99) 62%), #0a140e';
    }
}

// --- Элементы оверлея ---
const menuEl = document.getElementById('menu');
const lobbyOverlay = document.getElementById('lobbyOverlay');
const lobbyHome = document.getElementById('lobbyHome');
const lobbyParty = document.getElementById('lobbyParty');
const lobbyStatus = document.getElementById('lobbyStatus');
const lobbyCodeBig = document.getElementById('lobbyCodeBig');
const lobbyListEl = document.getElementById('lobbyList');
const lobbyStartBtn = document.getElementById('lobbyStartBtn');
const lobbyLeaveBtn = document.getElementById('lobbyLeaveBtn');
const lobbyCodeInput = document.getElementById('lobbyCodeInput');
const lobbyZoneRow = document.getElementById('lobbyZoneRow');
const lobbyBackBtn = document.getElementById('lobbyBackBtn');
const lobbyPreviewCanvas = document.getElementById('lobbyPreview');
const waveBannerEl = document.getElementById('waveBanner');
const waveBannerMain = document.getElementById('waveBannerMain');
const waveBannerSub = document.getElementById('waveBannerSub');
const startCountdownEl = document.getElementById('startCountdown');
const scNumEl = document.getElementById('scNum');
const deathWaveLine = document.getElementById('deathWaveLine');
const deathTitle = document.getElementById('deathTitle');
const deathSub = document.getElementById('deathSub');

const specialNames = {
    swarm: 'SWARM — many small fast mobs',
    tanks: 'TANKS — heavy high-rarity mobs',
    hunters: 'HUNTERS — aggressive chasers',
    boss: 'BOSS — a ladder boss has awakened',
    elite: 'ELITE — every mob is a variant (gold/ring/star/crescent)',
    fractal: 'BOSS — a ladder boss has awakened',
};

// --- Открытие/закрытие оверлея ---
function openLobbyOverlay() {
    menuEl.classList.add('lobby-open');
    document.body.classList.add('lobby-open');
    lobbyOverlay.style.display = 'flex';
    if (typeof renderMenuHotbar === 'function') renderMenuHotbar();
    refreshLobbyInfo();
    if (!lobbyPollTimer) lobbyPollTimer = setInterval(refreshLobbyInfo, 2000);
}
function closeLobbyOverlay() {
    menuEl.classList.remove('lobby-open');
    menuEl.classList.remove('party-view');
    document.body.classList.remove('lobby-open');
    lobbyOverlay.style.display = 'none';
    if (lobbyPollTimer) { clearInterval(lobbyPollTimer); lobbyPollTimer = null; }
}
if (lobbyBackBtn) lobbyBackBtn.addEventListener('click', closeLobbyOverlay);

// --- Опрос состояния ---
async function refreshLobbyInfo() {
    if (!authToken) return;
    try {
        const info = await apiCall('/api/lobby-info', { token: authToken });
        if (info.ok && info.inLobby) {
            lobbyWasInParty = true;
            setLobbyCode(info.code);
            renderLobbyParty(info);
        } else {
            // Пати распалось само (все вышли / reaper) — сообщаем, если это
            // не наш собственный выход (leave сбрасывает флаг до запроса).
            if (lobbyWasInParty) {
                lobbyWasInParty = false;
                if (errorBox) errorBox.textContent = 'The party was disbanded (everyone left)';
            }
            setLobbyCode('');
            renderLobbyHome();
        }
        refreshLobbyList();
    } catch (e) { /* сеть моргнула — retry в следующем тике */ }
}

function renderLobbyHome() {
    lobbyHome.style.display = 'flex';
    lobbyParty.style.display = 'none';
    menuEl.classList.remove('party-view');
    stopLobbyPreview();
    // Фоновых мобов нет на экране выбора пати.
    resetLobbyDecor();
    applyZoneBackground(0);
}

function renderLobbyParty(info) {
    lobbyHome.style.display = 'none';
    lobbyParty.style.display = 'flex';
    menuEl.classList.add('party-view');
    lobbyIsHost = !!info.isHost;
    lobbyCodeBig.textContent = 'CODE ' + (info.code || '------');
    lobbyStartBtn.style.display = lobbyIsHost ? '' : 'none';
    const wasStarted = lobbyMatchStarted;
    lobbyMatchStarted = !!info.started;
    if (lobbyMatchStarted) {
        lobbyStatus.textContent = 'Match running — good luck!';
    } else {
        lobbyStatus.textContent = lobbyIsHost
            ? 'Your party (' + ((info.playersData || []).length) + '/4). Pick the arena zone, then press ► Start.'
            : 'Waiting for the host to pick a zone and start…';
    }
    // Матч только что запустили (в т.ч. не-хост увидел через опрос) — таймер.
    if (lobbyMatchStarted && !wasStarted && !gameStarted) {
        leaveDuringMatch = false;
        showStartCountdown(5);
    } else if (lobbyMatchStarted && !gameStarted && !leaveDuringMatch
               && startCountdownEl.style.display === 'none') {
        // Самовосстановление: матч идёт, мы в пати, но отсчёта нет — переход
        // «started: false→true» был пропущен (сбой опроса, троттлинг вкладки,
        // перезагрузка страницы). Без этого ждущий игрок застревал навсегда.
        enterArena();
    }
    lobbyZone = (typeof info.zone === 'number') ? info.zone : 0;
    renderZoneSelector();
    applyZoneBackground(lobbyZone);
    startLobbyDecor(lobbyZone, info.zoneMobs || null);
    syncLobbyPreview(info.playersData || []);
}

// --- Фон и выбор зоны ---
function applyZoneBackground(zone) {
    lobbyOverlay.style.background = zoneBackgroundCss(zone);
}

function renderZoneSelector() {
    if (!lobbyIsHost) {
        const z = arenaZones[lobbyZone] || arenaZones[0];
        lobbyZoneRow.innerHTML = '<div class="zone-label" style="font-size:14px;color:#fff;font-weight:700;">Arena zone: ' +
            '<span style="color:' + z.css + '">' + z.name + '</span></div>';
        return;
    }
    lobbyZoneRow.innerHTML = '<div class="zone-label">Arena zone (host):</div>';
    arenaZones.forEach((z, i) => {
        const b = document.createElement('button');
        b.className = 'zone-btn' + (i === lobbyZone ? ' sel' : '');
        b.textContent = z.name;
        b.style.background = z.css;
        b.addEventListener('click', async () => {
            if (lobbyBusy) return;
            lobbyBusy = true;
            try {
                const res = await apiCall('/api/lobby-zone', { token: authToken, zone: i });
                if (res.ok) {
                    // Нормализуем: зона обязана быть числом (0 = Point).
                    lobbyZone = (typeof res.zone === 'number') ? res.zone : i;
                    renderZoneSelector();
                    applyZoneBackground(lobbyZone);
                    startLobbyDecor(lobbyZone, res.zoneMobs || null); // мобы фона сразу под новый биом
                } else errorBox.textContent = res.error || 'Failed to set zone';
            } finally { lobbyBusy = false; }
        });
        lobbyZoneRow.appendChild(b);
    });
}

// ============================================================
// Превью команды: персистентные слоты, плавные твины петалов,
// хотбар в 1-2 ряда. Никаких пересозданий на каждый опрос.
// ============================================================
let lobbyPreviewApp = null;
const lobbyPreviewSlots = new Map(); // nick -> slot state

// Ключ петала для сравнения (вид+редкость+цвет).
function petalKey(def) {
    return def ? (def.kind || 'oval') + ':' + (def.rarity || 0) + ':' + (def.color || 0) : 'null';
}

// Спрайт петала (настоящая форма игры) с ключом вида.
function makePetalSprite(def) {
    const g = new PIXI.Graphics();
    const col = (def && def.color != null) ? def.color : 0xecf0f1;
    const size = (def && def.size) ? def.size : 8;
    const kind = (def && def.kind) ? def.kind : 'oval';
    drawPetalShape(g, { color: col, size: size, kind: kind });
    g.__key = petalKey(def);
    return g;
}

function startLobbySync() {
    if (!lobbyPreviewApp) {
        lobbyPreviewApp = new PIXI.Application({
            view: lobbyPreviewCanvas,
            width: lobbyPreviewCanvas.width,
            height: lobbyPreviewCanvas.height,
            backgroundAlpha: 0,
            antialias: true,
        });
        lobbyPreviewApp.ticker.add(() => tickLobbyPreview(lobbyPreviewApp.ticker.deltaMS / 1000));
    }
}

function stopLobbyPreview() {
    if (lobbyPreviewApp) lobbyPreviewApp.stage.removeChildren();
    lobbyPreviewSlots.clear();
}

// Фолбэк пустого хотбара — 5 базовых Oval Point (как у новичка в игре).
function previewHotbar(m) {
    if (m.hotbar && m.hotbar.length) return m.hotbar.slice(0, 10);
    return [null, null, null, null, null];
}

function rangeIdx(n) { const a = []; for (let i = 0; i < n; i++) a.push(i); return a; }

// Раскладка хотбара: 1-5 — один ряд; 6-10 — два ряда по 5 (верхний чуть выше).
function hotbarLayout(count) {
    if (count <= 5) {
        return [{ y: -84, idx: rangeIdx(count) }];
    }
    return [
        { y: -102, idx: rangeIdx(5) },
        { y: -64,  idx: rangeIdx(count).slice(5) },
    ];
}

function createMemberSlot(m) {
    const st = {
        nick: m.nick,
        host: !!m.host,
        level: m.level || 0,
        bodyColor: m.nick === myNickname ? chosenColor : 0x54c468,
        hotbar: previewHotbar(m),
        orbitAngle: Math.random() * Math.PI * 2,
        petalSprites: [],
        petalAnim: [],
        hotbarCells: [],
        maxRar: 0,
    };
    const slot = new PIXI.Container();

    // Тело игрока в его цвете.
    const body = new PIXI.Graphics();
    body.beginFill(st.bodyColor);
    body.lineStyle(3, petalShade(st.bodyColor, 0.6));
    body.drawCircle(0, 0, 27);
    body.endFill();
    slot.addChild(body);

    st.orbit = new PIXI.Container();
    slot.addChild(st.orbit);
    st.hotbarRow = new PIXI.Container();
    slot.addChild(st.hotbarRow);
    st.label = new PIXI.Container();
    st.label.y = 50;
    slot.addChild(st.label);

    rebuildSlotHotbar(st);
    rebuildSlotLabel(st);
    rebuildSlotOrbit(st);

    st.container = slot;
    st.container.alpha = 0; // плавное появление в тикере
    lobbyPreviewApp.stage.addChild(slot);
    return st;
}

function rebuildSlotHotbar(st) {
    st.hotbarRow.removeChildren();
    st.hotbarCells = [];
    const items = st.hotbar;
    for (const row of hotbarLayout(items.length)) {
        const gap = 34;
        const startX = -(row.idx.length - 1) * gap / 2;
        row.idx.forEach((itemIdx, j) => {
            const p = items[itemIdx];
            const rarity = p ? Math.min(p.rarity || 0, rarityColors.length - 1) : 0;
            const rcol = rarityColors[rarity];
            const cell = new PIXI.Container();
            cell.x = startX + j * gap;
            cell.y = row.y;

            const frame = new PIXI.Graphics();
            frame.beginFill(0x000000, 0.35);
            frame.drawRoundedRect(-13, -13, 26, 26, 6);
            frame.endFill();
            frame.lineStyle(2.5, rcol, 0.95);
            frame.drawRoundedRect(-13, -13, 26, 26, 6);
            cell.addChild(frame);

            const sprite = makePetalSprite(p);
            sprite.scale.set(0.9);
            cell.addChild(sprite);

            st.hotbarRow.addChild(cell);
            st.hotbarCells[itemIdx] = { cell, frame, sprite, key: petalKey(p), a: 1, s: 0.9, pendingKey: null, pendingDef: null };
        });
    }
}

function rebuildSlotLabel(st) {
    st.label.removeChildren();
    let maxRar = 0;
    st.hotbar.forEach(p => { if (p && (p.rarity || 0) > maxRar) maxRar = p.rarity || 0; });
    st.maxRar = maxRar;
    const lvColor = rarityColors[maxRar];
    const nickT = new PIXI.Text(st.nick, { fill: 0xffffff, fontSize: 15, fontWeight: 'bold', dropShadow: 2, dropShadowColor: 0x000000, dropShadowAlpha: 0.8 });
    nickT.anchor.set(1, 0.5);
    const lvT = new PIXI.Text(String(st.level), { fill: lvColor, fontSize: 15, fontWeight: 'bold', dropShadow: 2, dropShadowColor: 0x000000, dropShadowAlpha: 0.8 });
    lvT.anchor.set(0, 0.5);
    const lblGap = 7, blockW = nickT.width + lblGap + lvT.width;
    nickT.x = -blockW / 2 + nickT.width;
    lvT.x = blockW / 2 - lvT.width;
    const pill = new PIXI.Graphics();
    pill.beginFill(0x0a0f18, 0.72);
    pill.lineStyle(2, lvColor, 0.9);
    pill.drawRoundedRect(-blockW / 2 - 14, -15, blockW + 28, 30, 15);
    pill.endFill();
    st.label.addChild(pill); st.label.addChild(nickT); st.label.addChild(lvT);
}

function rebuildSlotOrbit(st) {
    st.orbit.removeChildren();
    st.petalSprites = [];
    st.petalAnim = [];
    st.hotbar.forEach((p) => {
        const sprite = makePetalSprite(p);
        st.orbit.addChild(sprite);
        st.petalSprites.push(sprite);
        st.petalAnim.push({ a: 1, pendingKey: null, pendingDef: null });
    });
}

// Синхронизация с данными опроса: обновляем ТОЛЬКО изменившееся.
function syncLobbyPreview(members) {
    startLobbySync();
    const w = lobbyPreviewCanvas.width, h = lobbyPreviewCanvas.height;
    const n = Math.max(members.length, 1);

    // Удаляем ушедших.
    for (const [nick, st] of Array.from(lobbyPreviewSlots)) {
        if (!members.find(m => m.nick === nick)) {
            lobbyPreviewApp.stage.removeChild(st.container);
            lobbyPreviewSlots.delete(nick);
        }
    }

    members.forEach((m, idx) => {
        let st = lobbyPreviewSlots.get(m.nick);
        if (!st) st = lobbyPreviewSlots.set(m.nick, createMemberSlot(m)).get(m.nick);
        // Позиция по текущему индексу.
        st.container.x = w / (n + 1) * (idx + 1);
        st.container.y = h / 2 + 16;

        // Хотбар: свой берём из живого состояния клиента, обновляем плавно.
        const items = (m.nick === myNickname && myHotbars && myHotbars[myActiveBar])
            ? myHotbars[myActiveBar].slice(0, 10)
            : previewHotbar(m);

        if (items.length !== st.hotbar.length) {
            // Изменилось количество — перестраиваем ряды и орбиту целиком.
            st.hotbar = items;
            rebuildSlotHotbar(st);
            rebuildSlotOrbit(st);
            rebuildSlotLabel(st);
        } else {
            for (let i = 0; i < items.length; i++) {
                if (petalKey(items[i]) !== petalKey(st.hotbar[i])) {
                    st.hotbar[i] = items[i];
                    // Хотбар-ячейка: плавная замена.
                    const c = st.hotbarCells[i];
                    if (c) { c.pendingKey = petalKey(items[i]); c.pendingDef = items[i]; }
                    // Орбита: плавная замена.
                    const oa = st.petalAnim[i];
                    if (oa && oa.pendingKey === null) { oa.pendingKey = petalKey(items[i]); oa.pendingDef = items[i]; }
                }
            }
        }
        // Уровень/хост — только при изменении.
        const lvl = m.level || 0;
        if (st.level !== lvl || st.host !== !!m.host) {
            st.level = lvl; st.host = !!m.host;
            rebuildSlotLabel(st);
        }
    });
}

// Тикер: равномерное вращение орбит + твины замены петалов.
function tickLobbyPreview(dt) {
    for (const [, st] of lobbyPreviewSlots) {
        // Ячейки хотбара.
        for (const c of st.hotbarCells) {
            if (!c) continue;
            const targetA = c.pendingKey === null ? 1 : 0;
            if (c.pendingKey !== null) {
                c.a = Math.max(0, c.a - dt * 3.5);
                if (c.a <= 0.02) {
                    c.cell.removeChild(c.sprite);
                    c.sprite.destroy();
                    c.sprite = makePetalSprite(c.pendingDef);
                    c.sprite.scale.set(0.5);
                    c.cell.addChild(c.sprite);
                    c.key = c.pendingKey;
                    c.pendingKey = null; c.pendingDef = null;
                }
            } else {
                c.a = Math.min(1, c.a + dt * 3.5);
            }
            c.sprite.alpha = c.a;
            const targetS = c.pendingKey === null ? 0.9 : 0.4;
            c.s += (targetS - c.s) * Math.min(1, dt * 8);
            c.sprite.scale.set(c.s);
        }
        // Орбита: угол никогда не сбрасывается.
        st.orbitAngle += dt * 2.2;
        const k = st.petalSprites.length;
        for (let i = 0; i < st.petalSprites.length; i++) {
            let sprite = st.petalSprites[i];
            const anim = st.petalAnim[i];
            const a = st.orbitAngle + (2 * Math.PI / Math.max(k, 1)) * i;
            sprite.x = Math.cos(a) * 48;
            sprite.y = Math.sin(a) * 48;
            const targetA = anim.pendingKey === null ? 1 : 0;
            if (anim.pendingKey !== null) {
                anim.a = Math.max(0, anim.a - dt * 3.5);
                if (anim.a <= 0.02) {
                    const posIdx = st.orbit.getChildIndex(sprite);
                    const px = sprite.x, py = sprite.y;
                    st.orbit.removeChild(sprite);
                    sprite.destroy();
                    sprite = makePetalSprite(anim.pendingDef);
                    sprite.x = px; sprite.y = sprite.y0 = py;
                    sprite.scale.set(0.4);
                    st.orbit.addChildAt(sprite, posIdx);
                    st.petalSprites[i] = sprite;
                    anim.pendingKey = null; anim.pendingDef = null;
                }
            } else {
                anim.a = Math.min(1, anim.a + dt * 3.5);
            }
            sprite.alpha = anim.a;
            if (anim.a < 1) {
                const s = 0.4 + (1 - 0.4) * anim.a;
                sprite.scale.set(s);
            } else {
                sprite.scale.set(1);
            }
        }
        // Плавное появление слота.
        st.container.alpha = Math.min(1, st.container.alpha + dt * 2.5);
    }
}

// ============================================================
// Фоновые мобы биома: задержка ~1с, плотность по зоне, спавн
// порциями с вероятностями, пропорциональными весам зоны.
// ============================================================
let lobbyDecorApp = null;
let lobbyDecorMobs = [];
let lobbyDecorZone = -1;
let lobbyDecorPool = [];     // [{shape,color,size,weight}]
let lobbyDecorSpawnT = 0;    // таймер дозаполнения плотности
let lobbyDecorTarget = 18;   // целевая плотность (растёт с номером зоны)

function resetLobbyDecor() {
    lobbyDecorZone = -1;
    lobbyDecorPool = [];
    if (lobbyDecorApp) { lobbyDecorApp.stage.removeChildren(); lobbyDecorMobs = []; }
}

function startLobbyDecor(zone, zoneMobs) {
    // Сначала обновляем пул зоны (чтобы новые мобы были сразу нового биома).
    // Свежих данных нет — сбрасываем старый пул, чтобы не спавнить чужую зону.
    if (zoneMobs && zoneMobs.length) lobbyDecorPool = zoneMobs;
    else if (zone !== lobbyDecorZone) lobbyDecorPool = [];
    if (!lobbyDecorPool.length) {
        const z = arenaZones[zone] || arenaZones[0];
        lobbyDecorPool = [
            { shape: 'circle', color: z.color, size: 18, weight: 5 },
            { shape: 'square', color: z.color, size: 22, weight: 3 },
        ];
    }
    if (zone !== lobbyDecorZone) {
        // МГНОВЕННАЯ смена темы: старые мобы растворяются, новые сразу
        // появляются по всему экрану (без заезда с правого края).
        for (const d of lobbyDecorMobs) d.dying = true; // докрутятся с fade-out
        lobbyDecorZone = zone;
        lobbyDecorTarget = 16 + zone * 2; // плотность растёт к Chaos
        if (lobbyDecorApp) {
            const W = window.innerWidth, H = window.innerHeight;
            // Вся целевая плотность сразу — по всему экрану, без догона.
            for (let i = 0; i < lobbyDecorTarget; i++) {
                spawnDecorMob(Math.random() * W, 40 + Math.random() * (H - 120));
            }
        }
    }
    if (!lobbyDecorApp) {
        lobbyDecorApp = new PIXI.Application({
            view: document.getElementById('lobbyDecorCanvas'),
            width: window.innerWidth,
            height: window.innerHeight,
            backgroundAlpha: 0,
            antialias: true,
        });
        lobbyDecorApp.ticker.add(() => tickLobbyDecor(lobbyDecorApp.ticker.deltaMS / 1000));
        window.addEventListener('resize', () => {
            lobbyDecorApp.renderer.resize(window.innerWidth, window.innerHeight);
        });
    }
}

// Взвешенный выбор моба из пула зоны (веса = шансы спавна в зоне).
function pickDecorMob() {
    let total = 0;
    for (const d of lobbyDecorPool) total += d.weight;
    if (total <= 0) return lobbyDecorPool[0];
    let r = Math.random() * total;
    for (const d of lobbyDecorPool) {
        r -= d.weight;
        if (r < 0) return d;
    }
    return lobbyDecorPool[lobbyDecorPool.length - 1];
}

function spawnDecorMob(x, y) {
    const def = pickDecorMob();
    const c = new PIXI.Container();
    // Тёмная подложка-тень: мобы зоны видны даже когда цвет сливается
    // с фоном (зелёный Point на зелёном градиенте).
    const shadow = new PIXI.Graphics();
    shadow.beginFill(0x000000, 0.38);
    shadow.drawCircle(0, 0, Math.max(16, def.size * 1.35));
    shadow.endFill();
    c.addChild(shadow);
    const g = new PIXI.Graphics();
    drawMobShape(g, { shape: def.shape, color: def.color, size: def.size });
    c.addChild(g);
    const depth = 0.75 + Math.random() * 0.55;   // parallax: дальние мельче
    c.x = (x != null) ? x : window.innerWidth + 140;
    c.y = (y != null) ? y : 40 + Math.random() * (window.innerHeight - 120);
    c.alpha = 0;
    c.scale.set(depth);
    c.rotation = (Math.random() - 0.5) * 0.4;
    lobbyDecorApp.stage.addChild(c);
    lobbyDecorMobs.push({
        container: c,
        speed: (60 + Math.random() * 90) * depth, // дальние медленнее
        rot: (Math.random() - 0.5) * 0.6,
        // Мобы, распределённые по экрану при смене зоны, появляются сразу
        // полупрозрачными (без «пустого» кадра); заезжающие справа — с нуля.
        fade: (x != null) ? 0.6 : 0,
        wobbleA: Math.random() * Math.PI * 2,
        wobbleS: 0.6 + Math.random() * 1.2,
        baseY: c.y,
        dying: false,
    });
}

function tickLobbyDecor(dt) {
    if (lobbyDecorZone < 0 || !lobbyDecorPool.length) return;
    // Дозаполнение плотности: умирающие (старая зона) НЕ считаем —
    // иначе они блокируют спавн новых на время своего fade-out.
    let alive = 0;
    for (const d of lobbyDecorMobs) if (!d.dying) alive++;
    lobbyDecorSpawnT -= dt;
    if (lobbyDecorSpawnT <= 0 && alive < lobbyDecorTarget) {
        lobbyDecorSpawnT = 0.12 + Math.random() * 0.25;
        spawnDecorMob();
    }
    const W = window.innerWidth;
    for (let i = lobbyDecorMobs.length - 1; i >= 0; i--) {
        const d = lobbyDecorMobs[i];
        d.container.x -= d.speed * dt;
        d.container.rotation += d.rot * dt;
        // Лёгкое покачивание по вертикали — фон живой.
        d.wobbleA += d.wobbleS * dt;
        d.container.y = d.baseY + Math.sin(d.wobbleA) * 14;
        // Fade: быстрое появление и растворение «умирающих» (после смены зоны).
        if (d.dying) {
            d.fade -= dt * 7;
            if (d.fade <= 0) {
                lobbyDecorApp.stage.removeChild(d.container);
                d.container.destroy({ children: true });
                lobbyDecorMobs.splice(i, 1);
                continue;
            }
        } else {
            if (d.fade < 1) d.fade = Math.min(1, d.fade + dt * 5);
        }
        d.container.alpha = Math.max(0, d.fade) * 0.72;
        // Уплыл за левый край — на правый край с новой высотой (непрерывный поток).
        if (d.container.x < -160) {
            d.container.x = W + 160;
            d.baseY = 40 + Math.random() * (window.innerHeight - 120);
        }
    }
}

// --- Список публичных пати ---
function escapeHtml(s) {
    return String(s || '').replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
}

async function refreshLobbyList() {
    try {
        const res = await apiCall('/api/lobby-list', {});
        if (!res.ok || !Array.isArray(res.list)) return;
        lobbyListEl.innerHTML = '';
        if (!res.list.length) {
            lobbyListEl.innerHTML = '<div class="lc-empty">No open games — create your own!</div>';
            return;
        }
        for (const l of res.list) {
            const div = document.createElement('div');
            div.className = 'lobby-item' + (l.players >= l.max ? ' full' : '');
            const zc = arenaZones[l.zone] ? arenaZones[l.zone].css : '#2ecc71';
            div.innerHTML = '<span><b style="color:' + zc + '">●</b> ' + escapeHtml(l.host) + "'s party" +
                (l.started ? ' ⚔' : '') + '</span>' +
                '<span class="li-count">' + l.players + '/' + l.max + '</span>';
            if (l.players < l.max) {
                div.addEventListener('click', () => joinLobbyByCode(l.code));
            }
            lobbyListEl.appendChild(div);
        }
    } catch (e) { /* игнорируем */ }
}

async function joinLobbyByCode(code) {
    if (lobbyBusy) return;
    lobbyBusy = true;
    try {
        const res = await apiCall('/api/lobby-join', { token: authToken, code });
        if (res.ok) {
            setLobbyCode(res.code);
            errorBox.textContent = '';
            refreshLobbyInfo();
        } else {
            errorBox.textContent = res.error || 'Failed to join lobby';
            refreshLobbyList(); // список мог устареть — обновляем немедленно
        }
    } finally { lobbyBusy = false; }
}

// --- Кнопки (с busy-флагом против двойных кликов) ---
async function lobbyApiBusy(btn, fn) {
    if (lobbyBusy) return;
    lobbyBusy = true;
    if (btn) btn.disabled = true;
    try { await fn(); } finally {
        lobbyBusy = false;
        if (btn) btn.disabled = false;
    }
}

document.getElementById('lobbyCreateBtn').addEventListener('click', () => lobbyApiBusy(null, async () => {
    const res = await apiCall('/api/lobby-create', { token: authToken, public: true });
    if (res.ok) { setLobbyCode(res.code); refreshLobbyInfo(); }
    else errorBox.textContent = res.error || 'Failed to create lobby';
}));
document.getElementById('lobbyCreatePrivBtn').addEventListener('click', () => lobbyApiBusy(null, async () => {
    const res = await apiCall('/api/lobby-create', { token: authToken, public: false });
    if (res.ok) { setLobbyCode(res.code); refreshLobbyInfo(); }
    else errorBox.textContent = res.error || 'Failed to create lobby';
}));
document.getElementById('lobbyJoinBtn').addEventListener('click', () => {
    // БЕЗ lobbyApiBusy: joinLobbyByCode уже имеет собственный busy-флаг,
    // а вложенный вызов при lobbyBusy=true молча выходил — кнопка
    // «Join by Code» не работала вовсе (вход из списка работал).
    const code = (lobbyCodeInput.value || '').replace(/\D/g, '');
    if (code.length !== 6) { errorBox.textContent = 'Code is 6 digits'; return; }
    return joinLobbyByCode(code);
});
lobbyCodeInput.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') document.getElementById('lobbyJoinBtn').click();
});
lobbyLeaveBtn.addEventListener('click', () => lobbyApiBusy(lobbyLeaveBtn, async () => {
    lobbyWasInParty = false; // свой выход — без «пати распалось»
    await apiCall('/api/lobby-leave', { token: authToken });
    setLobbyCode('');
    refreshLobbyInfo();
}));
lobbyStartBtn.addEventListener('click', () => lobbyApiBusy(lobbyStartBtn, async () => {
    const res = await apiCall('/api/lobby-start', { token: authToken });
    if (!res.ok) errorBox.textContent = res.error || 'Failed to start';
    else {
        // Отсчёт видно сразу (в меню), не только тем, кто уже в арене по WS.
        showStartCountdown(5);
        refreshLobbyInfo();
    }
}));

// --- UI волн ---
function showWaveBanner(main, sub, ms) {
    waveBannerMain.textContent = main;
    waveBannerSub.textContent = sub || '';
    waveBannerEl.style.display = 'block';
    if (waveBannerTimer) clearTimeout(waveBannerTimer);
    waveBannerTimer = setTimeout(() => { waveBannerEl.style.display = 'none'; }, ms || 3000);
}

// Таймер «Starting in... N» — от Break секунд сервера. По окончании —
// автоматический вход всех в арену (спавн в центре карты).
function showStartCountdown(secs) {
    stopStartCountdown();
    const total = Math.max(1, Math.round(secs));
    let left = total;
    const tick = () => {
        if (left <= 0) {
            stopStartCountdown();
            // Все игроки автоматически заспавнятся посередине карты.
            if (!gameStarted && lobbyMatchStarted && typeof enterArena === 'function') {
                enterArena();
            }
            return;
        }
        scNumEl.textContent = left;
        startCountdownEl.style.display = 'block';
        left--;
    };
    tick();
    startCountdownTimer = setInterval(tick, 1000);
}
function stopStartCountdown() {
    if (startCountdownTimer) { clearInterval(startCountdownTimer); startCountdownTimer = null; }
    startCountdownEl.style.display = 'none';
}

function resetDeathToWaveDefaults() {
    // Авто-респавн на старте волны — убираем экран смерти.
    if (typeof hideDeath === 'function') hideDeath();
    if (deathTitle) deathTitle.textContent = 'You Died!';
    if (deathSub) deathSub.style.display = 'none';
    if (deathWaveLine) deathWaveLine.style.display = 'none';
}

function handleWaveMsg(msg) {
    switch (msg.state) {
        case 'pending':
            showStartCountdown(msg.break || 5);
            break;
        case 'start': {
            stopStartCountdown();
            resetDeathToWaveDefaults();
            const sub = specialNames[msg.special] || 'Clear all mobs';
            showWaveBanner('Wave ' + msg.num, sub, 3500);
            break;
        }
        case 'clear':
            showWaveBanner('Wave ' + msg.num + ' cleared!', 'Next wave soon', 2500);
            break;
        case 'gameover':
            stopStartCountdown();
            // Game Over: у всех только Return to Main Menu.
            showDeath();
            if (deathTitle) deathTitle.textContent = 'GAME OVER';
            if (deathSub) { deathSub.style.display = 'none'; }
            if (deathWaveLine) {
                let line = 'Your party reached wave ' + msg.num;
                if (msg.kills) line += ' · Kills: ' + msg.kills;
                if (msg.best && msg.best > msg.num) line += ' · Best: wave ' + msg.best;
                deathWaveLine.textContent = line;
                deathWaveLine.style.display = 'block';
            }
            break;
    }
}

// ============================================================
// Автоподключение к пати при загрузке страницы: если в localStorage остался
// код лобби (F5 во время матча / обрыв), сразу открываем экран пати — клиент
// увидит «Match running» и вернётся в арену по отсчёту, а не через Play вручную.
// ============================================================
if (authToken && getLobbyCode() && typeof openLobbyOverlay === 'function') {
    openLobbyOverlay();
}
