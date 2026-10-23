// editor.js — редактор стен зон (client/editor.html).
// Работает через /api/walls: GET — загрузить раскладку, POST — сохранить (админ).
(function () {
    'use strict';

    // ---- Масштаб: зона 8000x5000 рисуется на канвасе 800x500 ----
    const S = 0.1;
    const GRID = 100; // привязка в мировых единицах

    const cv = document.getElementById('cv');
    const ctx = cv.getContext('2d');

    // ---- Состояние ----
    let data = null;        // wallsDoc от сервера
    let cur = 0;            // активная зона
    let sel = -1;           // выбранная стена в активной зоне
    let drag = null;        // {mode:'draw'|'move'|'resize', ...}
    let dirty = false;

    const $ = (id) => document.getElementById(id);

    // ---- Утилиты ----
    function wx2c(x) { return x * S; }
    function wy2c(y) { return y * S; }
    function c2wx(cx) { return cx / S; }
    function c2wy(cy) { return cy / S; }
    function snap(v) {
        if (!$('snap').checked) return Math.round(v);
        return Math.round(v / GRID) * GRID;
    }
    function wallsOf() { return data.zones[cur]; }
    // Параметры активной зоны: у зон второго уровня Safe Zone и стены нет.
    function zoneMeta() {
        const zi = data.zoneSafeH && data.zoneSafeH[cur] != null ? cur : -1;
        return {
            w: data.zoneW, h: data.zoneH,
            safeH: zi >= 0 ? data.zoneSafeH[zi] : data.safeH,
            wallTh: zi >= 0 ? data.zoneWallTh[zi] : data.wallTh,
            wallGap: zi >= 0 ? data.zoneWallGap[zi] : data.wallGap,
        };
    }

    function status(msg) { $('status').textContent = msg || ''; }

    // ---- Загрузка раскладки ----
    async function loadWalls() {
        try {
            const r = await fetch('/api/walls');
            data = await r.json();
            data.zones = (data.zones || []).map(z => z || []);
            renderZones();
            sel = -1;
            dirty = false;
            render();
            syncFields();
            status('Layout loaded.');
        } catch (e) {
            status('Failed to load /api/walls: ' + e);
        }
    }

    // ---- Кнопки зон ----
    function renderZones() {
        const box = $('zones');
        box.innerHTML = '';
        data.zones.forEach((w, i) => {
            const b = document.createElement('button');
            b.className = 'zbtn' + (i === cur ? ' active' : '');
            const nm = data.names && data.names[i];
            b.textContent = nm ? nm + (i >= 8 ? ' (L2)' : '') : 'Zone ' + i;
            b.title = 'Zone ' + i;
            b.addEventListener('click', () => {
                cur = i; sel = -1;
                renderZones(); render(); syncFields();
            });
            box.appendChild(b);
        });
    }

    // ---- Отрисовка ----
    function render() {
        if (!data) return;
        const z = zoneMeta();
        ctx.clearRect(0, 0, cv.width, cv.height);

        // Фон зоны.
        ctx.fillStyle = '#0d121b';
        ctx.fillRect(0, 0, cv.width, cv.height);

        // Сетка 500x500.
        ctx.strokeStyle = 'rgba(160,190,220,0.08)';
        ctx.lineWidth = 1;
        for (let x = 0; x <= z.w; x += 500) { vLine(wx2c(x), 0, cv.height); }
        for (let y = 0; y <= z.h; y += 500) { hLine(wy2c(y), 0, cv.width); }

        // Mob Zone (выше разделительной стены) — лёгкая подсветка верхней части.
        if (z.safeH > 0) {
            const safeTop = z.h - z.safeH;
            const wallY = safeTop - z.wallTh;
            ctx.fillStyle = 'rgba(90,140,220,0.05)';
            ctx.fillRect(0, 0, cv.width, wy2c(wallY));

            // Safe Zone — светло-зелёная полоса внизу.
            ctx.fillStyle = 'rgba(125,255,176,0.10)';
            ctx.fillRect(0, wy2c(safeTop), cv.width, cv.height - wy2c(safeTop));

            // Разделительная стена (два сегмента + проход по центру).
            const cx = z.w / 2, half = z.wallGap / 2;
            ctx.fillStyle = '#31435c';
            ctx.fillRect(wx2c(0), wy2c(wallY), wx2c(cx - half), wy2c(z.wallTh));
            ctx.fillRect(wx2c(cx + half), wy2c(wallY), wx2c(z.w - (cx + half)), wy2c(z.wallTh));

            // Линии границы Safe Zone.
            ctx.strokeStyle = 'rgba(125,255,176,0.5)';
            ctx.lineWidth = 1;
            hLine(wy2c(wallY), 0, cv.width);
            hLine(wy2c(safeTop), 0, cv.width);
        }

        // Конфигурируемые стены.
        wallsOf().forEach((w, i) => drawWall(w, i, i === sel));
    }

    function vLine(x, y0, y1) { ctx.beginPath(); ctx.moveTo(x, y0); ctx.lineTo(x, y1); ctx.stroke(); }
    function hLine(y, x0, x1) { ctx.beginPath(); ctx.moveTo(x0, y); ctx.lineTo(x1, y); ctx.stroke(); }

    function drawWall(w, i, selected) {
        const x = wx2c(w.x), y = wy2c(w.y), ww = wx2c(w.w), wh = wy2c(w.h);
        ctx.fillStyle = selected ? '#e07068' : '#c0504a';
        ctx.fillRect(x, y, ww, wh);
        ctx.strokeStyle = selected ? '#ffffff' : '#8f3d38';
        ctx.lineWidth = selected ? 2 : 1;
        ctx.strokeRect(x, y, ww, wh);
        if (selected) {
            // Ручка изменения размера — правый нижний угол.
            ctx.fillStyle = '#ffffff';
            ctx.fillRect(x + ww - 7, y + wh - 7, 7, 7);
            ctx.strokeRect(x + ww - 7, y + wh - 7, 7, 7);
        }
        // Номер стены.
        ctx.fillStyle = 'rgba(255,255,255,0.9)';
        ctx.font = 'bold 11px Arial';
        ctx.fillText(String(i), x + 3, y + 12);
    }

    // ---- Поля выбранной стены ----
    function syncFields() {
        const w = sel >= 0 ? wallsOf()[sel] : null;
        $('fX').value = w ? w.x : '';
        $('fY').value = w ? w.y : '';
        $('fW').value = w ? w.w : '';
        $('fH').value = w ? w.h : '';
        $('btnApply').disabled = !w;
        $('btnDel').disabled = !w;
        $('jsonPre').textContent = JSON.stringify(data && data.zones, null, 2) || '';
    }

    function pickWallAt(wx, wy) {
        const ws = wallsOf();
        for (let i = ws.length - 1; i >= 0; i--) {
            const w = ws[i];
            if (wx >= w.x && wx < w.x + w.w && wy >= w.y && wy < w.y + w.h) return i;
        }
        return -1;
    }

    function cornerAt(wx, wy) {
        if (sel < 0) return false;
        const w = wallsOf()[sel];
        const padX = 60 / S, padY = 60 / S;
        return wx > w.x + w.w - padX && wy > w.y + w.h - padY;
    }

    // ---- Мышь ----
    function onDown(e) {
        if (!data) return;
        const p = canvPos(e);
        const wx = snap(c2wx(p.x)), wy = snap(c2wy(p.y));

        if (cornerAt(wx, wy)) {
            const w = wallsOf()[sel];
            drag = { mode: 'resize', startX: wx, startY: wy, ox: w.x, oy: w.y, ow: w.w, oh: w.h };
            return;
        }
        const i = pickWallAt(wx, wy);
        if (i >= 0) {
            sel = i;
            drag = { mode: 'move', startX: wx, startY: wy, ox: wallsOf()[i].x, oy: wallsOf()[i].y };
            css(); syncFields();
            return;
        }
        sel = -1;
        drag = { mode: 'draw', startX: wx, startY: wy, cx: wx, cy: wy };
        css(); syncFields();
    }

    function onMove(e) {
        if (!data || !drag) return;
        const p = canvPos(e);
        const wx = snap(c2wx(p.x)), wy = snap(c2wy(p.y));
        if (drag.mode === 'move') {
            const w = wallsOf()[sel];
            w.x = clamp(drag.ox + (wx - drag.startX), 0, data.zoneW - w.w);
            w.y = clamp(drag.oy + (wy - drag.startY), 0, data.zoneH - w.h);
            dirty = true;
        } else if (drag.mode === 'resize') {
            const w = wallsOf()[sel];
            w.w = Math.max(40, clamp(drag.ow + (wx - drag.startX), 40, data.zoneW - w.x));
            w.h = Math.max(40, clamp(drag.oh + (wy - drag.startY), 40, data.zoneH - w.y));
            dirty = true;
        } else if (drag.mode === 'draw') {
            drag.cx = wx; drag.cy = wy;
        }
        render(); syncFields();
    }

    function onUp() {
        if (drag && drag.mode === 'draw' && data) {
            const x0 = Math.min(drag.startX, drag.cx);
            const y0 = Math.min(drag.startY, drag.cy);
            const w = Math.abs(drag.cx - drag.startX);
            const h = Math.abs(drag.cy - drag.startY);
            if (w >= 40 && h >= 40) {
                wallsOf().push({ x: x0, y: y0, w: w, h: h });
                sel = wallsOf().length - 1;
                dirty = true;
                status('Wall added.');
            }
        }
        drag = null;
        render(); syncFields();
    }

    function canvPos(e) {
        const r = cv.getBoundingClientRect();
        const scx = cv.width / r.width;
        const scy = cv.height / r.height;
        return {
            x: (e.clientX - r.left) * scx,
            y: (e.clientY - r.top) * scy
        };
    }

    function clamp(v, lo, hi) { return Math.max(lo, Math.min(hi, v)); }

    // Стиль курсора при наведении.
    function css() {
        cv.style.cursor = drag ? 'grabbing' : 'crosshair';
    }

    // ---- Кнопки панели ----
    $('btnApply').addEventListener('click', () => {
        if (sel < 0) return;
        const w = wallsOf()[sel];
        w.x = clamp(parseFloat($('fX').value) || 0, 0, data.zoneW - 40);
        w.y = clamp(parseFloat($('fY').value) || 0, 0, data.zoneH - 40);
        w.w = clamp(parseFloat($('fW').value) || 40, 40, data.zoneW - w.x);
        w.h = clamp(parseFloat($('fH').value) || 40, 40, data.zoneH - w.y);
        dirty = true;
        render(); syncFields();
    });

    $('btnDel').addEventListener('click', () => {
        if (sel < 0) return;
        wallsOf().splice(sel, 1);
        sel = -1; dirty = true;
        render(); syncFields();
        status('Wall deleted.');
    });

    $('btnClear').addEventListener('click', () => {
        if (!confirm('Clear all walls in zone ' + cur + '?')) return;
        wallsOf().length = 0;
        sel = -1; dirty = true;
        render(); syncFields();
        status('Zone cleared.');
    });

    document.addEventListener('keydown', (e) => {
        if (e.key === 'Delete' && sel >= 0) $('btnDel').click();
    });

    cv.addEventListener('mousedown', onDown);
    window.addEventListener('mousemove', onMove);
    window.addEventListener('mouseup', onUp);
    cv.addEventListener('touchstart', (e) => { if (e.touches[0]) onDown({ clientX: e.touches[0].clientX, clientY: e.touches[0].clientY }); e.preventDefault(); }, { passive: false });
    cv.addEventListener('touchmove', (e) => { if (e.touches[0]) onMove({ clientX: e.touches[0].clientX, clientY: e.touches[0].clientY }); e.preventDefault(); }, { passive: false });
    cv.addEventListener('touchend', onUp);

    // ---- Авторизация админа + сохранение ----
    const LT = 'wallToken', LN = 'wallNick';
    $('aToken').value = localStorage.getItem(LT) || '';
    $('aNick').value = localStorage.getItem(LN) || '';

    $('btnLogin').addEventListener('click', async () => {
        const nick = $('aNick').value.trim();
        const pass = $('aPass').value;
        if (!nick || !pass) { status('Enter nick and password.'); return; }
        try {
            const r = await fetch('/api/login', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ nick, password: pass })
            });
            const d = await r.json();
            if (d.ok && d.token) {
                localStorage.setItem(LT, d.token);
                localStorage.setItem(LN, nick);
                $('aToken').value = d.token;
                $('aPass').value = '';
                status('Admin token received.');
            } else {
                status('Login error: ' + (d.error || 'no data'));
            }
        } catch (err) {
            status('Network unavailable: ' + err);
        }
    });

    $('btnSave').addEventListener('click', async () => {
        const token = $('aToken').value.trim();
        if (!token) { status('No admin token: sign in or paste a token.'); return; }
        try {
            const r = await fetch('/api/walls', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ token, zones: data.zones })
            });
            if (r.ok) {
                data = await r.json();
                dirty = false;
                render(); syncFields();
                status('Saved to walls.json. Players will see the walls after reloading the page.');
            } else {
                status('Save error: HTTP ' + r.status);
            }
        } catch (err) {
            status('Save error: ' + err);
        }
    });

    // ---- Старт ----
    loadWalls();
})();