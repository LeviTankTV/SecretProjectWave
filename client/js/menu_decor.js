// menu_decor.js — анимированный фон главного меню: летающие геометрические
// мобы («Shaper»-тема). Схема портирована из лобби SecretProjectWave
// (lobby.js): параллакс по глубине, синус-покачивание, вращение,
// бесконечный поток слева направо через перенос за правый край.
// Самодостаточно: только PIXI + палитра редкостей (rarities.js).
'use strict';

let menuDecorApp = null;
let menuDecorMobs = [];
let menuDecorSpawnT = 0;
let menuDecorStars = [];     // мерцающие звёзды заднего плана
// Режим декора: 'menu' — мобы летят слева направо (главное меню),
// 'auth' — шейпы медленно ПАДАЮТ сверху (экраны логина/регистрации).
// Орбита редкостей вокруг логотипа видна только в режиме 'menu'.
let menuDecorMode = 'menu';
// Целевые количества задаёт applyGfxQuality() (settings_input.js): на среднем
// уровне декор легче, на минимальном декор-канвас полностью останавливается.
let MENU_DECOR_TARGET = 18;
let MENU_STAR_TARGET = 70;
// Параметры шлейфа курсора: интервал спавна (мс) и максимум осколков за движение.
let ctMinInterval = 24, ctMaxShards = 5;

// menuDecorInitStars — статичное звёздное небо с мерцанием: точки трёх размеров,
// каждая дышит по своей фазе.
function menuDecorInitStars() {
    const W = window.innerWidth, H = window.innerHeight;
    for (let i = 0; i < MENU_STAR_TARGET; i++) {
        const g = new PIXI.Graphics();
        const r = 0.8 + Math.random() * 1.8;
        g.beginFill(Math.random() < 0.25 ? 0xffe9a8 : 0xdfeaff, 1);
        g.drawCircle(0, 0, r);
        g.endFill();
        g.x = Math.random() * W;
        g.y = Math.random() * H;
        g.alpha = 0.1 + Math.random() * 0.4;
        menuDecorApp.stage.addChildAt(g, 0);
        menuDecorStars.push({
            g,
            base: 0.12 + Math.random() * 0.4,
            ph: Math.random() * Math.PI * 2,
            sp: 0.4 + Math.random() * 1.4,
        });
    }
}

function tickMenuDecorStars(dt) {
    for (const s of menuDecorStars) {
        s.ph += s.sp * dt;
        s.g.alpha = s.base + (Math.sin(s.ph) * 0.5 + 0.5) * 0.45;
    }
}

// Пул декора: фигуры в цветах редкостей (rarities.js) — тематично для Shaper.
const MENU_DECOR_POOL = [
    { shape: 'circle',   ci: 0, size: 16, weight: 5 },
    { shape: 'triangle', ci: 2, size: 20, weight: 4 },
    { shape: 'square',   ci: 3, size: 20, weight: 4 },
    { shape: 'pentagon', ci: 4, size: 24, weight: 3 },
    { shape: 'hexagon',  ci: 5, size: 26, weight: 3 },
    { shape: 'star',     ci: 7, size: 26, weight: 2 },
    { shape: 'diamond',  ci: 6, size: 22, weight: 2 },
    { shape: 'spiral',   ci: 14, size: 26, weight: 1 },
];

function menuDecorPick() {
    let total = 0;
    for (const d of MENU_DECOR_POOL) total += d.weight;
    let r = Math.random() * total;
    for (const d of MENU_DECOR_POOL) {
        r -= d.weight;
        if (r < 0) return d;
    }
    return MENU_DECOR_POOL[0];
}

// menuDecorDrawShape — векторная фигура моба: тёмное тело, цветная обводка,
// светлый блик. g — PIXI.Graphics, def — элемент пула.
function menuDecorDrawShape(g, def) {
    const col = (typeof rarityColors !== 'undefined' && rarityColors[def.ci]) ? parseInt(rarityColors[def.ci].slice(1), 16) : 0x7be0ff;
    const s = def.size;
    const poly = (n, r, rot) => {
        const pts = [];
        for (let i = 0; i < n; i++) {
            const a = (rot || 0) + i * 2 * Math.PI / n;
            pts.push(Math.cos(a) * r, Math.sin(a) * r);
        }
        return pts;
    };
    g.lineStyle(Math.max(3, s * 0.16), col, 1);
    g.beginFill(0x12080b, 0.85);
    switch (def.shape) {
        case 'square': g.drawPolygon(poly(4, s, Math.PI / 4)); break;
        case 'triangle': g.drawPolygon(poly(3, s, -Math.PI / 2)); break;
        case 'pentagon': g.drawPolygon(poly(5, s, -Math.PI / 2)); break;
        case 'hexagon': g.drawPolygon(poly(6, s, 0)); break;
        case 'diamond': g.drawPolygon(poly(4, s, 0)); break;
        case 'star': {
            const pts = [];
            for (let i = 0; i < 10; i++) {
                const a = -Math.PI / 2 + i * Math.PI / 5;
                const rr = i % 2 ? s * 0.45 : s;
                pts.push(Math.cos(a) * rr, Math.sin(a) * rr);
            }
            g.drawPolygon(pts);
            break;
        }
        case 'spiral': {
            // «фрактал»: три вложенных повёрнутых квадрата
            g.drawPolygon(poly(4, s, Math.PI / 4));
            g.drawPolygon(poly(4, s * 0.62, Math.PI / 4 + 0.5));
            g.drawPolygon(poly(4, s * 0.3, Math.PI / 4 + 1));
            break;
        }
        default: g.drawCircle(0, 0, s);
    }
    g.endFill();
    // блик — маленький светлый кружок сверху-слева
    g.lineStyle(0);
    g.beginFill(0xffffff, 0.25);
    g.drawCircle(-s * 0.32, -s * 0.32, Math.max(2.5, s * 0.16));
    g.endFill();
}

function menuDecorSpawn(x, y) {
    // «Живой музей» (museum.js): в главном меню летят настоящие мобы из
    // галереи — открытые в цвете, неоткрытые тёмными силуэтами. Если музей
    // не готов (нет данных/иконок) — обычный абстрактный пул ниже.
    const fallingNow = menuDecorMode === 'auth';
    if (!fallingNow && typeof window.museumNextDef === 'function') {
        const md = window.museumNextDef();
        if (md && window.museumSpawnDecor && window.museumSpawnDecor(md, x, y)) return;
    }
    const def = menuDecorPick();
    const c = new PIXI.Container();
    const shadow = new PIXI.Graphics();
    shadow.beginFill(0x000000, 0.38);
    shadow.drawCircle(0, 0, Math.max(16, def.size * 1.35));
    shadow.endFill();
    c.addChild(shadow);
    const g = new PIXI.Graphics();
    menuDecorDrawShape(g, def);
    c.addChild(g);
    const depth = 0.75 + Math.random() * 0.55; // параллакс: дальние мельче
    const falling = menuDecorMode === 'auth';
    c.x = (x != null) ? x : (falling ? Math.random() * window.innerWidth : window.innerWidth + 140);
    c.y = (y != null) ? y : (falling ? -60 - Math.random() * 120 : 40 + Math.random() * (window.innerHeight - 120));
    c.alpha = 0;
    c.scale.set(depth);
    c.rotation = (Math.random() - 0.5) * 0.4;
    menuDecorApp.stage.addChild(c);
    menuDecorMobs.push({
        container: c,
        depth,
        // Режим падения: вертикальная скорость, горизонтальный дрейф малый.
        speed: falling
            ? (26 + Math.random() * 34) * depth
            : (60 + Math.random() * 90) * depth,
        rot: (Math.random() - 0.5) * 0.6,
        fade: (x != null) ? 0.6 : 0,
        wobbleA: Math.random() * Math.PI * 2,
        wobbleS: 0.6 + Math.random() * 1.2,
        baseY: c.y,
    });
}

// menuDecorFadeAbstract — попросить ВСЕ абстрактные фигуры раствориться.
// Зовёт музей (museum.js), когда проснулась зона: фон становится музейным, но
// без резкого «фигурки исчезли» на одном кадре.
function menuDecorFadeAbstract() {
    if (typeof menuDecorMobs === 'undefined') return;
    for (const d of menuDecorMobs) if (!d.museum) d.fadeOut = true;
}

function tickMenuDecor(dt) {
    if (typeof menuDecorEnabled !== 'undefined' && !menuDecorEnabled) return;
    let alive = menuDecorMobs.length;
    menuDecorSpawnT -= dt;
    if (menuDecorSpawnT <= 0 && alive < MENU_DECOR_TARGET) {
        menuDecorSpawnT = 0.12 + Math.random() * 0.25;
        menuDecorSpawn();
    }
    const W = window.innerWidth;
    const falling = menuDecorMode === 'auth';
    for (let i = menuDecorMobs.length - 1; i >= 0; i--) {
        const d = menuDecorMobs[i];
        d.wobbleA += d.wobbleS * dt;
        if (falling) {
            // Падение сверху вниз с мягким покачиванием по горизонтали.
            d.container.y += d.speed * dt;
            d.container.x += Math.sin(d.wobbleA) * 16 * dt;
            d.container.rotation += d.rot * 0.5 * dt;
        } else {
            d.container.x -= d.speed * dt;
            d.container.rotation += d.rot * dt;
            d.container.y = d.baseY + Math.sin(d.wobbleA) * 14;
        }
        if (d.fadeOut) {
            // Затухание уходящей абстрактной фигуры (museum.js ставит fadeOut,
            // когда проснулась зона): растворяется на ходу, а не исчезает кадром.
            // Ветка ВМЕСТО fade-in: иначе прибавка dt*5 перебивала бы убыль и
            // фигура так и не дошла бы до нуля.
            d.fade -= dt * 1.6;
            if (d.fade <= 0) {
                try { d.container.destroy({ children: true }); } catch (e) {}
                menuDecorMobs.splice(i, 1);
                continue;
            }
        } else if (d.fade < 1) {
            d.fade = Math.min(1, d.fade + dt * 5);
        }
        // глубина: дальние мобы бледнее (пелена глубины)
        d.container.alpha = Math.max(0, d.fade) * (0.35 + 0.35 * d.depth);
        // Кольцо-подсветка музейного моба под курсором (наведение, не карточка).
        if (d.ring) {
            d.hoverT = Math.min(1, Math.max(0, d.hoverT + (d.hoverT ? dt * 9 : -dt * 9)));
            d.ring.alpha = d.hoverT * 0.8;
            d.ring.scale.set(1 + 0.06 * Math.sin(d.wobbleA * 1.6));
            // Кольцо чуть крупнее самого моба + лёгкий «подъём» контейнера.
            d.container.scale.set(1 + 0.1 * d.hoverT);
        }
        if (falling) {
            if (d.container.y > window.innerHeight + 160) {
                d.container.y = -80;
                d.container.x = Math.random() * W;
                d.baseY = d.container.y;
            }
        } else if (d.container.x < -160) {
            if (d.museum) {
                // Музейный моб ушёл за край: заменяем его НОВЫМ видом из пула.
                // Раньше он просто переносился за правый край с той же
                // иконкой — сцена застывала, и в фоне бесконечно летал один
                // и тот же моб вместо всех открытых в галерее.
                try { d.container.destroy({ children: true }); } catch (e) {}
                menuDecorMobs.splice(i, 1);
                // Новый моб занимает освободившийся слот (справа за краем).
                menuDecorSpawn(undefined, undefined);
                continue;
            }
            d.container.x = W + 160;
            d.baseY = 40 + Math.random() * (window.innerHeight - 120);
        }
    }
}

// startMenuDecor — создать приложение (один раз) и засеять экран мобами.
// Вызывается при показе меню; при входе в игру — stopMenuDecor().
// Орбита редкостей вокруг логотипа: Point/Line/Triangle/Square/Pentagon/
// Hexagon/Fractal/Chaos — контурные фигуры в цветах зон на эллипсе.
const MENU_ORBIT_COLORS = [0x2ecc71, 0xf1c40f, 0xe67e22, 0xe74c3c, 0x3498db, 0x1abc9c, 0x9b59b6, 0x4a6d96]; // Chaos осветлён
// эллипс плотно вокруг текста SHAPER (высота чуть больше букв)
const MENU_ORBIT_RX = 275, MENU_ORBIT_RY = 64;
// Скорости орбиты (рад/с). Домножены на 2 — кольцо вокруг логотипа вращается
// вдвое быстрее во всех режимах: след за курсором (GAIN), потолок скорости
// (MAX_VEL), самовращение после простоя (IDLE_VEL) и разгон (ACCEL). Домножать
// надо все четыре, иначе после простоя кольцо уходило бы в старый темп, а
// разгон оставался бы вялым. Потолок затем поднят ещё ×1.5 (10.2), а разгон
// смягчён (7.5) — разгоняется кольцо чуть дольше, но до Нового потолка.
const ORBIT_GAIN = 4.4;     // след за курсором (было 2.2)
const ORBIT_MAX_VEL = 10.2; // потолок угловой скорости (было 3.4, затем 6.8)
const ORBIT_IDLE_VEL = 0.7; // самовращение после простоя (было 0.35)
const ORBIT_ACCEL = 7.5;    // рад/с², ограничение ускорения (было 5.5, затем 11)
let menuOrbit = null, menuOrbitAngle = 0, menuOrbitShapes = [];

// menuOrbitDraw — контурная фигура i-тира со свечением (полигон/круг/линия).
function menuOrbitDraw(g, tier) {
    const col = MENU_ORBIT_COLORS[tier];
    const s = 11 + tier * 2.2;
    g.beginFill(0x12080b, 0.8);
    g.lineStyle(7, col, 0.16); // псевдо-свечение: широкий полупрозрачный контур
    const poly = (n, r, rot) => {
        const pts = [];
        for (let i = 0; i < n; i++) {
            const a = (rot || 0) + i * 2 * Math.PI / n;
            pts.push(Math.cos(a) * r, Math.sin(a) * r);
        }
        return pts;
    };
    switch (tier) {
        case 0: g.lineStyle(0); g.beginFill(col, 0.15); g.drawCircle(0, 0, s * 0.55); g.endFill();
            g.lineStyle(3, col, 1); g.drawCircle(0, 0, s * 0.55); break;
        case 1: // Line — отрезок со скруглением
            g.lineStyle(0); g.endFill();
            g.lineStyle(7, col, 0.16); g.moveTo(-s, 0); g.lineTo(s, 0);
            g.lineStyle(3.5, col, 1); g.moveTo(-s, 0); g.lineTo(s, 0); break;
        case 2: g.drawPolygon(poly(3, s, -Math.PI / 2)); break;
        case 3: g.drawPolygon(poly(4, s, Math.PI / 4)); break;
        case 4: g.drawPolygon(poly(5, s, -Math.PI / 2)); break;
        case 5: g.drawPolygon(poly(6, s, 0)); break;
        case 6: // Fractal — три вложенных повёрнутых квадрата
            g.drawPolygon(poly(4, s, Math.PI / 4));
            g.drawPolygon(poly(4, s * 0.6, Math.PI / 4 + 0.5));
            g.drawPolygon(poly(4, s * 0.28, Math.PI / 4 + 1));
            break;
        default: { // Chaos — рваная звезда
            const pts = [];
            for (let i = 0; i < 14; i++) {
                const a = i * 2 * Math.PI / 14;
                const rr = s * (0.55 + (i % 2) * 0.5) * (0.85 + ((i * 37) % 10) / 33);
                pts.push(Math.cos(a) * rr, Math.sin(a) * rr);
            }
            g.drawPolygon(pts);
        }
    }
    g.endFill();
    g.lineStyle(3.5, col, 1);
    // повторный чистый контур поверх свечения (кроме Line — уже нарисован)
    if (tier !== 0 && tier !== 1) {
        switch (tier) {
            case 2: g.drawPolygon(poly(3, s, -Math.PI / 2)); break;
            case 3: g.drawPolygon(poly(4, s, Math.PI / 4)); break;
            case 4: g.drawPolygon(poly(5, s, -Math.PI / 2)); break;
            case 5: g.drawPolygon(poly(6, s, 0)); break;
            case 6:
                g.drawPolygon(poly(4, s, Math.PI / 4));
                g.drawPolygon(poly(4, s * 0.6, Math.PI / 4 + 0.5));
                g.drawPolygon(poly(4, s * 0.28, Math.PI / 4 + 1));
                break;
            default: {
                const pts = [];
                for (let i = 0; i < 14; i++) {
                    const a = i * 2 * Math.PI / 14;
                    const rr = s * (0.55 + (i % 2) * 0.5) * (0.85 + ((i * 37) % 10) / 33);
                    pts.push(Math.cos(a) * rr, Math.sin(a) * rr);
                }
                g.drawPolygon(pts);
            }
        }
    }
}

function menuOrbitInit() {
    menuOrbit = new PIXI.Container();
    menuOrbitShapes = [];
    for (let i = 0; i < 8; i++) {
        const c = new PIXI.Container();
        const g = new PIXI.Graphics();
        menuOrbitDraw(g, i);
        c.addChild(g);
        menuOrbit.addChild(c);
        menuOrbitShapes.push(c);
    }
    menuDecorApp.stage.addChild(menuOrbit);
    menuOrbitPlace();
}

// menuOrbitPlace — центр эллипса = фактический центр заголовка SHAPER.
function menuOrbitPlace() {
    if (!menuOrbit) return;
    const h1 = document.querySelector('.menu-logo');
    if (h1) {
        const r = h1.getBoundingClientRect();
        menuOrbit.x = r.left + r.width / 2;
        menuOrbit.y = r.top + r.height / 2;
    } else {
        menuOrbit.x = window.innerWidth / 2;
        menuOrbit.y = window.innerHeight * 0.33;
    }
}

function tickMenuOrbit(dt) {
    if (!menuOrbit) return;
    // Орбита «смотрит» на курсор: формирование доворачивается фигурой-point в
    // сторону курсора относительно центра логотипа. Модель СКОРОСТИ: желаемая
    // угловая скорость пропорциональна ошибке (быстрый живой след), но само
    // ускорение ограничено — при резком переводе курсора через центр орбита
    // НЕ разворачивается рывком: плавно тормозит до нуля и так же плавно
    // набирает ход в новую сторону. Мёртвой зоны нет — только буквальная
    // точка (2px) в центре, куда курсор попасть не может.
    // Курсор не двигались 5 с → кольцо переходит в самовращение.
    const mouseActive = (performance.now() - decorMouseMovedT) < (window.__menuOrbitIdleMs || 5000);
    let targetVel = menuOrbitIdleSpin; // самовращение после простоя
    if (mouseActive) {
        // самовращение плавно гасится, пока курсор жив
        menuOrbitIdleSpin = 0;
        const dx = decorMouse.x - menuOrbit.x, dy = decorMouse.y - menuOrbit.y;
        if (dx * dx + dy * dy > 4) {
            const target = Math.atan2(dy, dx);
            let diff = target - menuOrbitAngle;
            while (diff > Math.PI) diff -= 2 * Math.PI;
            while (diff < -Math.PI) diff += 2 * Math.PI;
            targetVel = diff * ORBIT_GAIN;           // пропорциональный след
            const maxVel = ORBIT_MAX_VEL;           // потолок скорости (рад/с)
            if (targetVel > maxVel) targetVel = maxVel;
            else if (targetVel < -maxVel) targetVel = -maxVel;
        } else {
            targetVel = 0;
        }
    } else {
        // медленный разгон самовращения после простоя (~2 с до полной скорости)
        menuOrbitIdleSpin += (ORBIT_IDLE_VEL - menuOrbitIdleSpin) * Math.min(1, dt * 0.5);
    }
    // Ограничение углового УСКОРЕНИЯ: скорость меняется плавно — это и даёт
    // «остановился и заново набирает ход» при развороте на 180°.
    const accel = ORBIT_ACCEL; // рад/с²
    const dv = targetVel - menuOrbitVel;
    const maxDv = accel * dt;
    if (dv > maxDv) menuOrbitVel += maxDv;
    else if (dv < -maxDv) menuOrbitVel -= maxDv;
    else menuOrbitVel = targetVel;
    menuOrbitAngle += menuOrbitVel * dt;
    for (let i = 0; i < menuOrbitShapes.length; i++) {
        const a = menuOrbitAngle + i * Math.PI / 4;
        menuOrbitShapes[i].x = Math.cos(a) * MENU_ORBIT_RX;
        menuOrbitShapes[i].y = Math.sin(a) * MENU_ORBIT_RY;
        menuOrbitShapes[i].rotation += dt * 0.4;
    }
}
let decorMouse = { x: -1e9, y: -1e9 };
let decorMouseMovedT = -1e9; // время последнего движения мыши (performance.now)
let menuOrbitIdleSpin = 0;   // скорость самовращения после простоя (плавный разгон)
let menuOrbitVel = 0;        // текущая угловая скорость (рад/с)
document.addEventListener('mousemove', (e) => {
    decorMouse.x = e.clientX; decorMouse.y = e.clientY;
    decorMouseMovedT = performance.now();
});

// decorFadeOut / decorFadeIn — плавное гашение и возврат декор-канваса (меню и
// реалма). Раньше канвас прятался через display:none мгновенно, и при нажатии
// Start фон обрывался рывком, пока панель перехода ещё ехала по экрану
// (замечание владельца: «фон мгновенно пропадает»). Рендер при этом всё равно
// останавливается сразу (stop() выше по коду) — тает последний кадр, так что
// на самом заметном участке анимации FPS не тратится.
const DECOR_FADE_MS = 260;
window.decorFadeOut = function (el) {
    if (!el) return;
    if (el.__decorFadeT) { clearTimeout(el.__decorFadeT); el.__decorFadeT = null; }
    if (el.style.display === 'none') return;
    el.style.transition = 'opacity ' + DECOR_FADE_MS + 'ms ease';
    el.style.opacity = '0';
    el.__decorFadeT = setTimeout(() => {
        el.__decorFadeT = null;
        // За время таяния канвас могли показать заново (старт декора) — тогда
        // display:none уже не нужен, иначе канвас исчезнет из-под новой сцены.
        if (el.style.opacity === '0') el.style.display = 'none';
    }, DECOR_FADE_MS);
};
window.decorFadeIn = function (el) {
    if (!el) return;
    if (el.__decorFadeT) { clearTimeout(el.__decorFadeT); el.__decorFadeT = null; }
    el.style.transition = 'none';
    el.style.display = 'block';
    el.style.opacity = '1';
};

function startMenuDecor(mode) {
    // Режим декора: 'menu' (главное меню, летят слева) или 'auth'
    // (логин/регистрация, падают сверху). Смена режима пересеивает сцену.
    const m = (mode === 'auth') ? 'auth' : 'menu';
    const modeChanged = (m !== menuDecorMode);
    menuDecorMode = m;
    const cv = document.getElementById('menuDecorCanvas');
    if (!cv) return;
    // Минимальная графика: декор-канвас не запускаем вовсе (applyGfxQuality
    // вызывает stopMenuDecor — канвас и виньетка скрыты, тикер стоит).
    if (typeof gfxQuality !== 'undefined' && gfxQuality === 0) { stopMenuDecor(); return; }
    window.decorFadeIn(cv);
    const vg = document.getElementById('menuVignette');
    if (vg) window.decorFadeIn(vg);
    // миникарта — игровой элемент, в меню ей не место
    const mm = document.getElementById('minimap');
    if (mm) mm.style.display = 'none';
    if (!menuDecorApp) {
        menuDecorApp = new PIXI.Application({
            view: cv,
            width: window.innerWidth,
            height: window.innerHeight,
            backgroundAlpha: 0,
            antialias: true,
        });
        menuDecorApp.ticker.add(() => {
            const dt = menuDecorApp.ticker.deltaMS / 1000;
            tickMenuDecor(dt);
            tickMenuDecorStars(dt);
            tickMenuOrbit(dt);
        });
        menuDecorInitStars();
        menuOrbitInit();
        window.addEventListener('resize', () => {
            menuDecorApp.renderer.resize(window.innerWidth, window.innerHeight);
            menuOrbitPlace();
        });
        for (let i = 0; i < MENU_DECOR_TARGET; i++) {
            if (menuDecorMode === 'auth') {
                // Падающие шейпы: равномерно по всему экрану (не только сверху),
                // чтобы сцена была живой сразу, без ожидания «дождя».
                menuDecorSpawn(Math.random() * window.innerWidth, Math.random() * window.innerHeight);
            } else {
                menuDecorSpawn(Math.random() * window.innerWidth, 40 + Math.random() * (window.innerHeight - 120));
            }
        }
    } else {
        menuDecorApp.start();
        // сцена была очищена при выходе — пересоздать звёзды и орбиту
        if (!menuDecorStars.length) menuDecorInitStars();
        if (!menuOrbit) menuOrbitInit();
        menuOrbitPlace();
        // Смена режима (меню ↔ auth): старые мобы двигались в другую сторону —
        // вычищаем и сеем заново в новом режиме.
        if (modeChanged) {
            for (const d of menuDecorMobs) d.container.destroy();
            menuDecorMobs = [];
        }
        // Мобы могли быть вычищены при остановке (минимальная графика) —
        // мгновенно засеиваем экран заново, не дожидаясь потока спавна.
        if (menuDecorMobs.length === 0 && MENU_DECOR_TARGET > 0 &&
            (typeof menuDecorEnabled === 'undefined' || menuDecorEnabled)) {
            for (let i = 0; i < MENU_DECOR_TARGET; i++) {
                if (menuDecorMode === 'auth') {
                    menuDecorSpawn(Math.random() * window.innerWidth, Math.random() * window.innerHeight);
                } else {
                    menuDecorSpawn(Math.random() * window.innerWidth, 40 + Math.random() * (window.innerHeight - 120));
                }
            }
        }
    }
    // Орбита редкостей вокруг логотипа — только в главном меню (на логине/
    // регистрации логотипа нет, орбита центрируется на пустоту).
    if (menuOrbit) menuOrbit.visible = (menuDecorMode === 'menu');
}

function stopMenuDecor() {
    if (menuDecorApp) {
        menuDecorApp.stop();
        // убрать все фигуры со сцены: иначе последний кадр
        // с летающими мобами остаётся видимым поверх игры
        menuDecorApp.stage.removeChildren();
        menuDecorMobs = [];
        menuDecorStars = [];
        menuOrbit = null;
        menuOrbitShapes = [];
    }
    // Канвас гасим ВСЕГДА, даже если приложение ещё не создано, но плавно:
    // мгновенный display:none обрывал фон меню рывком на входе в игру.
    const cv = document.getElementById('menuDecorCanvas');
    if (cv) window.decorFadeOut(cv);
    const vg = document.getElementById('menuVignette');
    if (vg) window.decorFadeOut(vg);
    // Шлейф курсора здесь НЕ чистим: его убирает startBiomeWipe при входе
    // в игру (во время плашки шлейф скрыт целиком — body.booting).
    // вернуться в игру — миникарта снова управляется своими настройками.
    // НЕ безусловно: start() зовёт stopMenuDecor на середине биом-свитфа,
    // и сброс display:'' возвращал радар в ещё не накрытый правый нижний
    // угол (до markCovered его прятал только syncTicker). applyMinimap
    // уважает гейт свипа (__bwipeReveal === false) и настройку миникарты.
    if (typeof applyMinimap === 'function') {
        applyMinimap();
    } else if (window.__bwipeReveal !== false) {
        const mm = document.getElementById('minimap');
        if (mm) mm.style.display = '';
    }
}

// Автостарт при загрузке страницы (меню/логин активны сразу);
// дальше включением/выключением рулит showScreen().
if (typeof PIXI !== 'undefined' && document.getElementById('menuDecorCanvas')) {
    startMenuDecor();
}

// ===== Шлейф за курсором: «осколки петалей» вместо золотых искр =====
// Тематика та же, что у фонового декора: мини-фигуры в палитре редкостей
// (rarities.js). Каждый осколок отрывается от курсора, летит против движения,
// крутится и растворяется. Живёт только на экранах меню (menuDecorApp).
const CT_SHAPES = [
    'polygon(50% 0%, 100% 100%, 0% 100%)',                        // треугольник
    'polygon(50% 0%, 100% 50%, 50% 100%, 0% 50%)',                // ромб
    'polygon(28% 0%, 72% 0%, 100% 28%, 100% 72%, 72% 100%, 28% 100%, 0% 72%, 0% 28%)', // квадрат
    'polygon(50% 0%, 98% 35%, 79% 95%, 21% 95%, 2% 35%)',         // пятиугольник
    'polygon(25% 2%, 75% 2%, 98% 50%, 75% 98%, 25% 98%, 2% 50%)', // шестиугольник
    'polygon(50% 0%, 61% 39%, 100% 50%, 61% 61%, 50% 100%, 39% 61%, 0% 50%, 39% 39%)', // звезда
];
// Индексы редкостей и веса повторяют MENU_DECOR_POOL — шлейф в той же палитре.
const CT_RARITY_IDX = [0, 2, 3, 4, 5, 7, 6, 14];
const CT_RARITY_W = [6, 4, 4, 3, 3, 2, 2, 1];

let ctLast = 0, ctX = 0, ctY = 0, ctStarted = false;
let ctColorOverride = null; // цвет шлейфа от экранов (<=0 сбрасывает)

// setCursorTrailColor(cssColor) — перекрасить шлейф (экран Choose Realm).
// null или не-строка сбрасывает оверрайд: шлейф снова берёт цвета из палитры
// редкостей. Раньше оверрайд жил в window.__ctColorOverride и НИГДЕ не
// сбрасывался при уходе с экрана, поэтому однотонный цвет «залипал» и в
// главном меню (после F5 он подхватывался из сохранённого выбора реалма).
function setCursorTrailColor(css) {
    ctColorOverride = (typeof css === 'string' && css) ? css : null;
}
function getCursorTrailColor() { return ctColorOverride; }

function ctSpawnShard(layer, x, y, backX, backY) {
    let col, ci = 0;
    const ov = ctColorOverride;
    if (ov) {
        col = ov;
    } else {
        let r = Math.random() * CT_RARITY_W.reduce((a, b) => a + b, 0);
        for (; ci < CT_RARITY_W.length; ci++) {
            r -= CT_RARITY_W[ci];
            if (r <= 0) break;
        }
        if (ci >= CT_RARITY_IDX.length) ci = 0;
        col = (typeof rarityColors !== 'undefined' && rarityColors[CT_RARITY_IDX[ci]])
            ? rarityColors[CT_RARITY_IDX[ci]] : '#7be0ff';
    }
    // Во время загрузки мира (body.booting) осколки крупнее — красивее смотреть.
    const booting = document.body.classList.contains('booting');
    const sz = booting ? 10 + Math.random() * 7 : 6 + Math.random() * 5;
    const d = document.createElement('div');
    d.className = 'ct-shard';
    d.style.left = x + 'px';
    d.style.top = y + 'px';
    d.style.width = sz + 'px';
    d.style.height = sz + 'px';
    d.style.margin = (-sz / 2) + 'px 0 0 ' + (-sz / 2) + 'px';
    d.style.background = col;
    d.style.clipPath = CT_SHAPES[ci % CT_SHAPES.length];
    d.style.setProperty('--glow', col);
    layer.appendChild(d);
    // Отрыв: чем быстрее шла мышь, тем дальше улетает осколок назад.
    const kick = 0.35 + Math.random() * 0.5;
    const dx = backX * kick + (Math.random() - 0.5) * 10;
    const dy = backY * kick + (Math.random() - 0.5) * 10;
    const rot = (Math.random() - 0.5) * 90;
    const spin = (Math.random() - 0.5) * 320;
    const anim = d.animate([
        { transform: 'rotate(' + rot + 'deg) scale(.25)', opacity: 0 },
        {
            transform: 'translate(' + (dx * 0.4).toFixed(1) + 'px,' + (dy * 0.4).toFixed(1) + 'px) ' +
                'rotate(' + (rot + spin * 0.5).toFixed(1) + 'deg) scale(1)',
            opacity: 0.95, offset: 0.3,
        },
        {
            transform: 'translate(' + dx.toFixed(1) + 'px,' + dy.toFixed(1) + 'px) ' +
                'rotate(' + (rot + spin).toFixed(1) + 'deg) scale(.3)',
            opacity: 0,
        },
    ], { duration: 620 + Math.random() * 220, easing: 'cubic-bezier(.2,.75,.3,1)' });
    anim.onfinish = () => d.remove();
}

// Шаг ленты и потолок на одно событие мыши.
//
// Живой отчёт 7: при быстром движении в следе оставались пустые линии.
// Причина — раскладка «n = round(dist / 22), но не больше ctMaxShards»:
// на длинном отрезке между двумя событиями мыши осколков ставилось пять, и
// между ними оставались дыры в десятки пикселей. Теперь позиции считаются от
// НАКОПЛЕННОЙ дистанции с шагом CT_STEP — лента непрерывна на любой скорости.
// Если один рывок даёт больше CT_MAX_PER_EVENT осколков, растягивается ШАГ на
// всю дистанцию: осколки встают равномерно реже, но без разрывов.
let ctPending = 0;
const CT_STEP = 18;
const CT_MAX_PER_EVENT = 12;

document.addEventListener('mousemove', (e) => {
    const layer = document.getElementById('cursorTrail');
    if (!layer || !menuDecorApp) return;
    // Минимальная графика: шлейф выключен полностью.
    if (typeof gfxQuality !== 'undefined' && gfxQuality === 0) return;
    if (!ctStarted) { ctX = e.clientX; ctY = e.clientY; ctStarted = true; ctPending = 0; return; }
    const dx = e.clientX - ctX, dy = e.clientY - ctY;
    const dist = Math.hypot(dx, dy);
    if (dist < 1) return;
    // Ограничитель частоты остаётся (это защита от лишней работы на быстрой
    // мыши), но он больше НЕ рвёт ленту: пропущенные пиксели копятся в
    // ctPending и отрисовываются при первом же принятом событии.
    const minIv = (typeof ctMinInterval === 'number') ? ctMinInterval : 24;
    const ready = performance.now() - ctLast >= minIv;
    ctX = e.clientX; ctY = e.clientY;
    if (!ready) { ctPending += dist; return; }
    ctLast = performance.now();
    const ux = dx / dist, uy = dy / dist;
    const push = Math.min(1, dist / 60) * 26;
    const total = ctPending + dist;
    let step = CT_STEP;
    let count = Math.floor(total / step);
    const cap = (typeof ctMaxShards === 'number') ? Math.max(1, ctMaxShards) : CT_MAX_PER_EVENT;
    const maxN = Math.min(cap, CT_MAX_PER_EVENT);
    if (count > maxN) { count = maxN; step = total / count; }
    // Остаток, не дотянувший до целого шага, переносим на следующее событие —
    // он и делает ленту сплошной вместо ритмичных пропусков.
    ctPending = total - count * step;
    for (let k = count; k >= 1; k--) {
        const back = total - k * step;
        if (back < 0) continue;
        ctSpawnShard(layer, e.clientX - ux * back, e.clientY - uy * back, -ux * push, -uy * push);
    }
});

// Сворачивание/переключение окна: события мыши не приходят, и после возврата
// курсор может стоять в другом месте. Сбрасываем опорную точку, чтобы первый
// после возврата mousemove не рисовал линию осколков от старой позиции к
// новой, а шлейф начинался заново с текущего места курсора. То же при выходе
// курсора за пределы окна.
window.addEventListener('blur', () => { ctStarted = false; ctPending = 0; });
document.addEventListener('visibilitychange', () => {
    if (document.hidden) { ctStarted = false; ctPending = 0; }
});
document.addEventListener('mouseleave', () => { ctStarted = false; ctPending = 0; });

// След живёт только в меню: при выходе в игру очищаем осколки с экрана.
function clearCursorTrail() {
    const layer = document.getElementById('cursorTrail');
    if (layer) layer.replaceChildren();
    ctX = 0; ctY = 0; ctStarted = false; ctPending = 0;
}
