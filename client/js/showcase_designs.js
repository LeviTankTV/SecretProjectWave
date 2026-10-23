/* ============================================================================
   ВИТРИНА TestRoom — ДИЗАЙНЫ МОКАПОВ СЛОТОВ 24-77
   ============================================================================

   ЗДЕСЬ лежит «картинка» каждого мокапа. Имена/цвета — в
   server/showcase_slots.go, а сервер и клиент сами их подхватывают.

   ФОРМАТ ОДНОГО СЛОТА (это БЛОКИ 24-77):

       SHOWCASE_DESIGNS[24] = {
           name: "Bud",                            // информационно; в игре имя берётся из showcase_slots.go
           draw: function (g, r, col, t) {
               // --- код рисования в НЕГО (см. контракт ниже) ---
           }
       };

   Что получает draw:
     g   — PIXI.Graphics, уже пустой (clear + без детей), центр моба в (0,0).
     r   — радиус (≈42): тело должно жить внутри r; лучи/ауры можно до ~1.4*r.
     col — базовый цвет моба (0xRRGGBB) из реестра; дизайн может строить
           свою палитру вокруг него (шлёмай в оттенки: shade(col, amt)).
     t   — время в секундах (performance.now()/1000) — вся анимация от него.

   ДОСТУПНО ГЛОБАЛЬНО: PIXI (v7), shade(hex, amt), starPath(g, n, rOuter,
   rInner, phase), arcPath(g, r, a0, a1, steps). Можно создавать детями g
   временные PIXI.Container/Graphics — каждый кадр они очищаются АВТОМАТИЧЕСКИ.

   ПРАВИЛА (обязательно):
     - Ссылаться на PIXI/shade/starPath/arcPath можно ТОЛЬКО внутри draw,
       не на верхнем уровне файла.
     - Первые строки draw должны быть:  g.lineStyle(0);
     - Без setTimeout / fetch / внешних картинок. Только Graphics-примитивы.
     - Рисуется КАЖДЫЙ КАДР (много мобов × 60fps) — держи код лёгким, без
       создания тяжёлых фильтров в цикле кадров.
     - Анимация плавная: вращение/орбиты/пульс через cos/sin(t), без рывков
       и без отсылок к разным t в одном месте.

   УДАЛИТЬ моба = удалить его блок (слот станет пустой заглушкой).
   ========================================================================== */

window.SHOWCASE_DESIGNS = window.SHOWCASE_DESIGNS || {};

// --- Заглушка: слот без блока — пульсирующее «пустое место» --------------
function showcasePlaceholder(g, r, col, t) {
    g.lineStyle(0);
    const dark = shade(col, -55);
    const light = shade(col, 45);
    const pulse = 0.5 + 0.5 * Math.sin(t * 2);
    // внешняя рамка-пунктир (8 штрихов, медленно вращается)
    g.lineStyle(2, col, 0.45 + 0.2 * pulse);
    for (let i = 0; i < 8; i++) {
        const a = i * Math.PI / 4 + t * 0.5;
        const ax = Math.cos(a) * r * 1.1;
        const ay = Math.sin(a) * r * 1.1;
        g.moveTo(Math.cos(a) * r * 0.95 + ax * 0, Math.sin(a) * r * 0.95);
        g.arc(ax, ay, r * 0.12, a - 0.6, a + 0.6);
    }
    g.lineStyle(3, 0x000000, 0.5);
    g.beginFill(dark, 0.6 + 0.25 * pulse);
    g.drawCircle(0, 0, r * (0.5 + 0.08 * pulse));
    g.endFill();
    g.beginFill(light, 0.9);
    g.drawCircle(0, 0, r * 0.18);
    g.endFill();
}

// --- Диспетчер слота: запускает блок из реестра или заглушку ---------------
function drawShowcaseSlot(slot, g, size, col) {
    g.lineStyle(0);
    g.clear();
    while (g.children.length) g.removeChildAt(0);
    const t = performance.now() / 1000;
    const d = SHOWCASE_DESIGNS[slot];
    if (d && typeof d.draw === 'function') {
        try {
            d.draw(g, size, col, t);
            return;
        } catch (e) {
            console.error("Showcase slot", slot, "failed in draw:", e);
            g.clear();
            while (g.children.length) g.removeChildAt(0);
        }
    }
    showcasePlaceholder(g, size, col, t);
}

/* ============================================================================
   ДИЗАЙНЫ СЛОТОВ 24-77 (54 шт.) — перенесены из showcase_new.txt 1:1.
   ========================================================================== */

// 24 — Bud: цветок с лепестками и ядром.
SHOWCASE_DESIGNS[24] = {
    name: "Bud",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -55);
        const light = shade(col, 55);
        const pulse = 0.5 + 0.5 * Math.sin(t * 1.7);

        g.beginFill(light, 0.2);
        g.drawCircle(0, 0, r * (1.08 + 0.05 * pulse));
        g.endFill();

        const N = 5;
        for (let i = 0; i < N; i++) {
            const a = (i / N) * Math.PI * 2 - Math.PI / 2;
            g.lineStyle(2.5, 0x000000, 0.55);
            g.beginFill(light, 0.9);
            g.drawCircle(Math.cos(a) * r * 0.5, Math.sin(a) * r * 0.5, r * 0.32);
            g.endFill();
        }

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(col, 1);
        g.drawCircle(0, 0, r * (0.42 + 0.04 * pulse));
        g.endFill();

        g.lineStyle(0);
        g.beginFill(dark, 0.25);
        g.drawCircle(0, r * 0.1, r * 0.3);
        g.endFill();

        g.beginFill(0xffffff, 0.6);
        g.drawCircle(-r * 0.12, -r * 0.12, r * 0.1);
        g.endFill();
    }
};

// 25 — Marble: спиральная намотка по шару.
SHOWCASE_DESIGNS[25] = {
    name: "Marble",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -55);
        const light = shade(col, 60);
        const pulse = 0.5 + 0.5 * Math.sin(t * 1.5);

        g.beginFill(light, 0.2);
        g.drawCircle(0, 0, r * (1.05 + 0.05 * pulse));
        g.endFill();

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(col, 1);
        g.drawCircle(0, 0, r * 0.82);
        g.endFill();

        g.lineStyle(3, light, 0.85);
        const N = 30;
        for (let i = 0; i <= N; i++) {
            const u = i / N;
            const a = u * Math.PI * 3 + t * 0.4;
            const rr = u * r * 0.72;
            const x = Math.cos(a) * rr;
            const y = Math.sin(a) * rr;
            if (i === 0) g.moveTo(x, y); else g.lineTo(x, y);
        }
        g.lineStyle(0);

        g.beginFill(dark, 0.18);
        g.drawCircle(0, r * 0.25, r * 0.55);
        g.endFill();

        g.beginFill(0xffffff, 0.55);
        g.drawCircle(-r * 0.28, -r * 0.28, r * 0.15);
        g.endFill();
    }
};

// 26 — Lancer: копьё с хвостом.
SHOWCASE_DESIGNS[26] = {
    name: "Lancer",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const light = shade(col, 55);
        const dark = shade(col, -50);
        const pulse = 0.5 + 0.5 * Math.sin(t * 2.5);

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(dark, 0.9);
        g.drawRect(-r*1.0, -r*0.06, r*1.6, r*0.12);
        g.endFill();

        g.beginFill(col, 1);
        g.drawPolygon([r*1.1, 0, r*0.6, -r*0.28, r*0.6, r*0.28]);
        g.endFill();

        g.lineStyle(0);
        g.beginFill(light, 0.8);
        g.drawPolygon([r*0.95, 0, r*0.65, -r*0.1, r*0.65, r*0.1]);
        g.endFill();

        g.beginFill(light, 0.4 + 0.4 * pulse);
        g.drawCircle(-r*0.95, 0, r*0.14 * (0.8 + 0.4 * pulse));
        g.endFill();
    }
};

// 27 — Ribbon: волнообразная лента.
SHOWCASE_DESIGNS[27] = {
    name: "Ribbon",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const light = shade(col, 60);
        const N = 20;
        const phase = t * 1.2;

        const flat = [];
        for (let i = 0; i <= N; i++) {
            const u = i / N;
            const x = -r + u * 2 * r;
            const yTop = -r*0.32 + Math.sin(u*Math.PI*2 + phase) * r*0.18;
            flat.push(x, yTop);
        }
        for (let i = N; i >= 0; i--) {
            const u = i / N;
            const x = -r + u * 2 * r;
            const yBot = r*0.32 + Math.sin(u*Math.PI*2 + phase) * r*0.18;
            flat.push(x, yBot);
        }
        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(col, 1);
        g.drawPolygon(flat);
        g.endFill();

        g.lineStyle(2, light, 0.9);
        for (let i = 0; i <= N; i++) {
            const u = i / N;
            const x = -r + u * 2 * r;
            const y = Math.sin(u*Math.PI*2 + phase) * r*0.18;
            if (i === 0) g.moveTo(x, y); else g.lineTo(x, y);
        }
        g.lineStyle(0);
    }
};

// 28 — Whip: плеть с утолщением к рукояти.
SHOWCASE_DESIGNS[28] = {
    name: "Whip",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const light = shade(col, 55);
        const phase = t * 2.2;
        const N = 16;
        const pts = [];
        for (let i = 0; i <= N; i++) {
            const u = i / N;
            const x = -r*0.9 + u * r*1.8;
            const y = Math.sin(u*Math.PI*1.6 + phase) * r*0.4;
            const w = r*0.22 * (1 - u*0.7);
            pts.push({x, y, w});
        }
        const flat = [];
        for (const p of pts) flat.push(p.x, p.y - p.w);
        for (let i = pts.length - 1; i >= 0; i--) flat.push(pts[i].x, pts[i].y + pts[i].w);
        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(col, 1);
        g.drawPolygon(flat);
        g.endFill();

        g.lineStyle(0);
        const hi = [];
        for (const p of pts) hi.push(p.x, p.y - p.w*0.4);
        for (let i = pts.length - 1; i >= 0; i--) hi.push(pts[i].x, pts[i].y + pts[i].w*0.4);
        g.beginFill(light, 0.55);
        g.drawPolygon(hi);
        g.endFill();

        const tip = pts[pts.length-1];
        g.beginFill(0xffffff, 0.9);
        g.drawCircle(tip.x, tip.y, r*0.09);
        g.endFill();
    }
};

// 29 — Javelin: дротик с древком.
SHOWCASE_DESIGNS[29] = {
    name: "Javelin",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const light = shade(col, 55);
        const dark = shade(col, -55);

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(dark, 0.7);
        g.drawRect(-r*0.9, -r*0.07, r*1.8, r*0.14);
        g.endFill();

        g.beginFill(col, 1);
        g.drawPolygon([r*1.05, 0, r*0.65, -r*0.22, r*0.7, 0, r*0.65, r*0.22]);
        g.endFill();

        g.beginFill(dark, 1);
        g.drawCircle(-r*0.85, 0, r*0.14);
        g.endFill();

        g.lineStyle(0);
        g.beginFill(light, 0.85);
        g.drawCircle(-r*0.9, -r*0.05, r*0.05);
        g.endFill();
    }
};

// 30 — Vector: стрелка-указатель с входящими линиями.
SHOWCASE_DESIGNS[30] = {
    name: "Vector",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const light = shade(col, 60);
        const pulse = 0.5 + 0.5 * Math.sin(t * 2.8);

        for (let i = -1; i <= 1; i++) {
            const y = i * r * 0.22;
            const len = r * 0.5 + pulse * r * 0.25;
            g.lineStyle(2, col, 0.55);
            g.moveTo(-r*0.9, y);
            g.lineTo(-r*0.9 + len, y);
        }

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(col, 1);
        g.drawPolygon([r*0.9, 0, r*0.2, -r*0.4, -r*0.4, -r*0.18, -r*0.4, r*0.18, r*0.2, r*0.4]);
        g.endFill();

        g.lineStyle(0);
        g.beginFill(light, 0.6);
        g.drawPolygon([r*0.75, 0, r*0.2, -r*0.25, -r*0.2, -r*0.08, -r*0.2, r*0.08, r*0.2, r*0.25]);
        g.endFill();
    }
};

// 31 — Bolt: молния.
SHOWCASE_DESIGNS[31] = {
    name: "Bolt",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const light = shade(col, 60);
        const pulse = 0.5 + 0.5 * Math.sin(t * 3);

        g.beginFill(col, 0.18 + 0.15 * pulse);
        g.drawCircle(0, 0, r*0.9);
        g.endFill();

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(col, 1);
        g.drawPolygon([
            r*0.1, -r*0.9,
            -r*0.4, -r*0.1,
            r*0.0, -r*0.1,
            -r*0.25, r*0.9,
            r*0.55, r*0.0,
            r*0.1, r*0.0,
            r*0.35, -r*0.9
        ]);
        g.endFill();

        g.lineStyle(0);
        g.beginFill(light, 0.75 + 0.2 * pulse);
        g.drawCircle(0, 0, r*0.12);
        g.endFill();
    }
};

// 32 — Darter: Рыба-фуга, ноздри и рот.
SHOWCASE_DESIGNS[32] = {
    name: "Darter",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const light = shade(col, 55);
        const dark = shade(col, -55);
        const pulse = 0.5 + 0.5 * Math.sin(t * 3.5);

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(dark, 1);
        g.drawPolygon([-r*0.3, -r*0.1, -r*0.9, -r*0.5, -r*0.7, 0]);
        g.endFill();
        g.beginFill(dark, 1);
        g.drawPolygon([-r*0.3, r*0.1, -r*0.9, r*0.5, -r*0.7, 0]);
        g.endFill();

        g.beginFill(col, 1);
        g.drawPolygon([r*0.9, 0, 0, -r*0.25, -r*0.5, 0, 0, r*0.25]);
        g.endFill();

        g.lineStyle(0);
        g.beginFill(light, 0.85);
        g.drawCircle(r*0.4, -r*0.05, r*0.07);
        g.endFill();

        g.beginFill(0xffffff, pulse * 0.85);
        g.drawCircle(-r*0.4, 0, r*0.06);
        g.endFill();
    }
};

// 33 — Trail: клинья-след.
SHOWCASE_DESIGNS[33] = {
    name: "Trail",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const light = shade(col, 60);
        const dark = shade(col, -50);
        const phase = t * 2;

        for (let i = 0; i < 5; i++) {
            const u = i / 4;
            const x = -r*0.9 + u * r*1.8;
            const s = 0.4 + u * 0.6;
            const y = Math.sin(phase + i*0.5) * r * 0.06;
            const w = r * 0.05 * s;
            const h = r * 0.25 * s;
            const l = r * 0.28 * s;
            g.lineStyle(2, 0x000000, 0.5);
            g.beginFill(u > 0.5 ? col : dark, 0.5 + u * 0.5);
            g.drawPolygon([x+l, y, x, y-h, x-l, y, x, y+h]);
            g.endFill();
        }

        g.lineStyle(0);
        g.beginFill(light, 0.95);
        g.drawCircle(r*0.9, Math.sin(phase+2) * r*0.06, r*0.1);
        g.endFill();
    }
};

// 34 — CometTail: комета с пучком хвостов. (В пасте блок был оборван —
// тело дописано в том же стиле.)
SHOWCASE_DESIGNS[34] = {
    name: "CometTail",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -55);
        const light = shade(col, 60);
        const pulse = 0.5 + 0.5 * Math.sin(t * 1.8);
        const tailLen = r * (0.8 + 0.15 * Math.sin(t * 2.5));

        for (let i = 0; i < 5; i++) {
            const a = -0.4 + i * 0.2;
            const len = tailLen * (0.6 + 0.4 * Math.sin(t * 3 + i));
            g.lineStyle(3, 0x000000, 0.4);
            g.beginFill(dark, 0.3 + 0.1 * i);
            g.moveTo(0, 0);
            g.lineTo(-len * Math.cos(a), -len * Math.sin(a));
            g.lineTo(-len * Math.cos(a) * 0.8, -len * Math.sin(a) * 0.8 - r * 0.1);
            g.closePath();
            g.endFill();
        }

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(col, 1);
        g.drawCircle(r * 0.3, 0, r * (0.34 + 0.05 * pulse));
        g.endFill();

        g.lineStyle(0);
        g.beginFill(light, 0.9);
        g.drawCircle(r * 0.42, -r * 0.08, r * 0.14);
        g.endFill();

        g.beginFill(0xffffff, 0.7);
        g.drawCircle(r * 0.22, -r * 0.16, r * 0.09);
        g.endFill();
    }
};

// 35 — SpikeLance: копьё с шипами и пульсирующим ядром.
SHOWCASE_DESIGNS[35] = {
    name: "SpikeLance",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -60);
        const light = shade(col, 70);
        const pulse = 0.5 + 0.5 * Math.sin(t * 3);
        const thrust = Math.sin(t * 4) * r * 0.1;

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(col, 1);
        g.moveTo(r * 0.9 + thrust, 0);
        g.lineTo(-r * 0.3 + thrust, -r * 0.15);
        g.lineTo(-r * 0.7 + thrust, 0);
        g.lineTo(-r * 0.3 + thrust, r * 0.15);
        g.closePath();
        g.endFill();

        for (let i = 0; i < 3; i++) {
            const x = -r * 0.2 + i * r * 0.25 + thrust;
            g.beginFill(dark, 0.7);
            g.moveTo(x, -r * 0.1);
            g.lineTo(x - r * 0.08, -r * 0.35);
            g.lineTo(x + r * 0.05, -r * 0.1);
            g.closePath();
            g.endFill();

            g.beginFill(dark, 0.7);
            g.moveTo(x, r * 0.1);
            g.lineTo(x - r * 0.08, r * 0.35);
            g.lineTo(x + r * 0.05, r * 0.1);
            g.closePath();
            g.endFill();
        }

        g.beginFill(light, 0.9);
        g.drawCircle(r * 0.4 + thrust, 0, r * (0.1 + 0.04 * pulse));
        g.endFill();
    }
};

// 36 — CometShard: кометный осколок с падающими искрами.
SHOWCASE_DESIGNS[36] = {
    name: "CometShard",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const light = shade(col, 65);
        const pulse = 0.5 + 0.5 * Math.sin(t * 1.8);

        for (let i = 0; i < 5; i++) {
            const k = i / 4;
            const x = -r * 0.2 - k * r * 1.1;
            const y = (i % 2 === 0 ? 1 : -1) * k * r * 0.35 + Math.sin(t * 3 - i) * r * 0.05;
            const sz = r * (0.22 - k * 0.15);
            g.lineStyle(3, 0x000000, 0.55);
            g.beginFill(i === 0 ? col : light, 1 - k * 0.5);
            g.moveTo(x, y - sz);
            g.lineTo(x + sz * 0.8, y);
            g.lineTo(x, y + sz);
            g.lineTo(x - sz * 0.8, y);
            g.closePath();
            g.endFill();
        }

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(col, 1);
        g.drawCircle(r * 0.4, 0, r * 0.55 + r * 0.05 * pulse);
        g.endFill();

        g.beginFill(0xffffff, 0.8);
        g.drawCircle(r * 0.25, -r * 0.1, r * 0.16);
        g.endFill();

        g.beginFill(light, 0.55);
        g.drawCircle(r * 0.55, -r * 0.18, r * 0.08);
        g.endFill();
    }
};

// 37 — BarbCore: шипастое ядро с вращающимися зубцами.
SHOWCASE_DESIGNS[37] = {
    name: "BarbCore",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -60);
        const light = shade(col, 55);
        const pulse = 0.5 + 0.5 * Math.sin(t * 3.5);
        const rot = t * 0.5;

        const n = 12;
        for (let i = 0; i < n; i++) {
            const a = rot + i * Math.PI * 2 / n;
            const len = r * (0.7 + 0.35 * pulse * (i % 2));
            g.lineStyle(2.5, 0x000000, 0.55);
            g.beginFill(i % 3 === 0 ? light : col, 1);
            g.drawPolygon([
                Math.cos(a - 0.12) * r * 0.5, Math.sin(a - 0.12) * r * 0.5,
                Math.cos(a) * len, Math.sin(a) * len,
                Math.cos(a + 0.12) * r * 0.5, Math.sin(a + 0.12) * r * 0.5
            ]);
            g.endFill();
        }

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(dark, 1);
        g.drawPolygon([0, -r * 0.5, r * 0.45, r * 0.35, -r * 0.45, r * 0.35]);
        g.endFill();

        g.lineStyle(0);
        g.beginFill(light, 1);
        g.drawCircle(0, 0, r * (0.12 + 0.05 * pulse));
        g.endFill();
    }
};

// 38 — Trigon: тройной треугольник-матрешка.
SHOWCASE_DESIGNS[38] = {
    name: "Trigon",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -60);
        const light = shade(col, 55);
        const pulse = 0.5 + 0.5 * Math.sin(t * 2);
        const rot = t * 0.6;

        for (let i = 2; i >= 0; i--) {
            const scale = 1 - i * 0.28;
            const rr = r * scale;
            const a = rot * (i + 1);
            const c = Math.cos(a), s = Math.sin(a);
            const pts = [
                [0, -rr],
                [rr * 0.87, rr * 0.5],
                [-rr * 0.87, rr * 0.5]
            ];
            g.lineStyle(3, 0x000000, 0.55);
            g.beginFill(i === 0 ? col : (i === 1 ? shade(col, -20) : dark), 1);
            g.moveTo(pts[0][0] * c - pts[0][1] * s, pts[0][0] * s + pts[0][1] * c);
            for (let j = 1; j < 3; j++) {
                g.lineTo(pts[j][0] * c - pts[j][1] * s, pts[j][0] * s + pts[j][1] * c);
            }
            g.closePath();
            g.endFill();
        }

        g.lineStyle(0);
        g.beginFill(light, 1);
        g.drawCircle(0, 0, r * (0.1 + 0.04 * pulse));
        g.endFill();
        g.beginFill(0xffffff, 0.7);
        g.drawCircle(-r * 0.03, -r * 0.03, r * 0.04);
        g.endFill();
    }
};

// 39 — TriShuriken: трёхлучевой сюрикен (двухслойное вращение).
SHOWCASE_DESIGNS[39] = {
    name: "TriShuriken",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const light = shade(col, 40);
        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(col, 1);
        starPath(g, 3, r * 1.0, r * 0.3, t * 1.5);
        g.endFill();
        g.beginFill(light, 0.6);
        starPath(g, 3, r * 0.6, r * 0.2, -t * 2.0);
        g.endFill();
        g.beginFill(0x000000, 0.2);
        g.drawCircle(0, 0, r * 0.15);
        g.endFill();
    }
};

// 40 — SwiftArrow: стрела с хвостовой воронкой.
SHOWCASE_DESIGNS[40] = {
    name: "SwiftArrow",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const trail = Math.sin(t * 5.0) * 0.1;
        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(col, 1);
        g.moveTo(0, -r * 0.9);
        g.lineTo(r * 0.4, r * 0.4);
        g.lineTo(0, r * 0.2);
        g.lineTo(-r * 0.4, r * 0.4);
        g.closePath();
        g.endFill();
        g.beginFill(shade(col, -40), 0.6);
        g.moveTo(0, r * 0.2);
        g.lineTo(r * 0.3, r * 0.8 + trail * r);
        g.lineTo(0, r * 0.6);
        g.lineTo(-r * 0.3, r * 0.8 + trail * r);
        g.closePath();
        g.endFill();
    }
};

// 41 — Triarch: вращающийся тройной треугольник.
SHOWCASE_DESIGNS[41] = {
    name: "Triarch",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -60);
        const light = shade(col, 55);
        const pulse = 0.5 + 0.5 * Math.sin(t * 2.1);
        const rot = t * 0.45;

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(dark, 0.22);
        g.drawPolygon([
            Math.cos(rot) * r * 0.92, Math.sin(rot) * r * 0.92,
            Math.cos(rot + Math.PI * 2 / 3) * r * 0.92, Math.sin(rot + Math.PI * 2 / 3) * r * 0.92,
            Math.cos(rot + Math.PI * 4 / 3) * r * 0.92, Math.sin(rot + Math.PI * 4 / 3) * r * 0.92
        ]);
        g.endFill();

        g.lineStyle(2, light, 0.8);
        g.beginFill(col, 0.95);
        g.drawPolygon([
            Math.cos(rot - 0.12) * r * 0.72, Math.sin(rot - 0.12) * r * 0.72,
            Math.cos(rot + Math.PI * 2 / 3 - 0.12) * r * 0.72, Math.sin(rot + Math.PI * 2 / 3 - 0.12) * r * 0.72,
            Math.cos(rot + Math.PI * 4 / 3 - 0.12) * r * 0.72, Math.sin(rot + Math.PI * 4 / 3 - 0.12) * r * 0.72
        ]);
        g.endFill();

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(light, 0.9);
        g.drawCircle(0, 0, r * (0.19 + pulse * 0.045));
        g.endFill();

        g.beginFill(0xffffff, 0.55);
        g.drawCircle(-r * 0.07, -r * 0.09, r * 0.07);
        g.endFill();
    }
};

// 42 — Spikewheel: тройная звезда-шестерня.
SHOWCASE_DESIGNS[42] = {
    name: "Spikewheel",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -65);
        const light = shade(col, 60);
        const rot = t * 1.25;
        const pulse = 0.5 + 0.5 * Math.sin(t * 2.7);

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(dark, 0.2);
        starPath(g, 12, r * 0.95, r * 0.48, rot);
        g.endFill();

        g.lineStyle(2, light, 0.9);
        g.beginFill(col, 0.92);
        starPath(g, 8, r * 0.72, r * 0.48, -rot * 0.7);
        g.endFill();

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(dark, 0.75);
        starPath(g, 3, r * (0.5 + pulse * 0.04), r * 0.27, t * 0.7);
        g.endFill();

        g.beginFill(light, 0.9);
        g.drawCircle(0, 0, r * 0.13);
        g.endFill();
    }
};

// 43 — FangCore: клыкастое ядро.
SHOWCASE_DESIGNS[43] = {
    name: "FangCore",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -70);
        const light = shade(col, 55);
        const rot = t * 0.8;
        const pulse = 0.5 + 0.5 * Math.sin(t * 2.5);

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(dark, 0.28);
        starPath(g, 6, r * 0.95, r * 0.5, rot);
        g.endFill();

        g.lineStyle(2, light, 0.85);
        g.beginFill(col, 0.95);
        starPath(g, 3, r * 0.76, r * 0.42, rot + Math.PI / 2);
        g.endFill();

        g.beginFill(dark, 0.8);
        g.drawCircle(0, 0, r * 0.27);
        g.endFill();

        g.lineStyle(2, light, 1);
        g.beginFill(light, 0.85);
        g.drawCircle(0, 0, r * (0.1 + pulse * 0.05));
        g.endFill();
    }
};

// 44 — ShardBurst: осколочный взрыв.
SHOWCASE_DESIGNS[44] = {
    name: "ShardBurst",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -65);
        const light = shade(col, 60);
        const pulse = 0.5 + 0.5 * Math.sin(t * 3);
        const rot = t * 0.55;

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(dark, 0.25);
        starPath(g, 9, r * (0.92 + pulse * 0.06), r * 0.42, rot);
        g.endFill();

        g.lineStyle(2, light, 0.8);
        g.beginFill(col, 0.95);
        starPath(g, 3, r * 0.72, r * 0.3, rot + Math.PI / 2);
        g.endFill();

        g.beginFill(light, 0.3);
        g.drawCircle(0, 0, r * (0.3 + pulse * 0.1));
        g.endFill();

        g.beginFill(0xffffff, 0.6);
        g.drawCircle(-r * 0.09, -r * 0.1, r * 0.075);
        g.endFill();
    }
};

// 45 — Trident: трезубец (вращается через g.rotation).
SHOWCASE_DESIGNS[45] = {
    name: "Trident",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -65);
        const light = shade(col, 65);
        const rot = t * 0.38;

        g.rotation = rot;

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(dark, 0.28);
        g.moveTo(0, -r * 0.92);
        g.lineTo(r * 0.22, -r * 0.25);
        g.lineTo(r * 0.7, -r * 0.7);
        g.lineTo(r * 0.48, r * 0.08);
        g.lineTo(0, r * 0.88);
        g.lineTo(-r * 0.48, r * 0.08);
        g.lineTo(-r * 0.7, -r * 0.7);
        g.lineTo(-r * 0.22, -r * 0.25);
        g.closePath();
        g.endFill();

        g.lineStyle(2, light, 0.95);
        g.beginFill(col, 0.96);
        g.moveTo(0, -r * 0.72);
        g.lineTo(r * 0.16, -r * 0.1);
        g.lineTo(r * 0.42, -r * 0.5);
        g.lineTo(r * 0.3, r * 0.14);
        g.lineTo(0, r * 0.66);
        g.lineTo(-r * 0.3, r * 0.14);
        g.lineTo(-r * 0.42, -r * 0.5);
        g.lineTo(-r * 0.16, -r * 0.1);
        g.closePath();
        g.endFill();

        g.beginFill(light, 0.9);
        g.drawCircle(0, r * 0.02, r * 0.13);
        g.endFill();
    }
};

// 46 — ApexCore: пирамида со светящимся ядром.
SHOWCASE_DESIGNS[46] = {
    name: "ApexCore",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -70);
        const light = shade(col, 65);
        const pulse = 0.5 + 0.5 * Math.sin(t * 2.6);

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(dark, 0.25);
        g.drawPolygon([
            0, -r * 0.95,
            r * 0.78, r * 0.58,
            -r * 0.78, r * 0.58
        ]);
        g.endFill();

        g.lineStyle(2, light, 0.9);
        g.beginFill(col, 0.92);
        g.drawPolygon([
            0, -r * 0.68,
            r * 0.55, r * 0.42,
            -r * 0.55, r * 0.42
        ]);
        g.endFill();

        g.beginFill(0x000000, 0.35);
        g.drawCircle(0, r * 0.12, r * 0.3);
        g.endFill();

        g.beginFill(light, 0.95);
        g.drawCircle(0, r * 0.08, r * (0.1 + pulse * 0.06));
        g.endFill();

        g.beginFill(0xffffff, 0.5);
        g.drawCircle(-r * 0.04, r * 0.02, r * 0.05);
        g.endFill();
    }
};

// 47 — InfernoTri: огненный тройной треугольник.
SHOWCASE_DESIGNS[47] = {
    name: "InfernoTri",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -75);
        const light = shade(col, 70);
        const pulse = 0.5 + 0.5 * Math.sin(t * 3.1);
        const rot = t * 0.42;

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(dark, 0.32);
        starPath(g, 12, r * (0.9 + pulse * 0.08), r * 0.72, rot);
        g.endFill();

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(col, 0.95);
        starPath(g, 3, r * 0.68, r * 0.34, rot + Math.PI / 2);
        g.endFill();

        g.lineStyle(2, light, 0.95);
        g.beginFill(light, 0.25);
        starPath(g, 3, r * 0.45, r * 0.22, -rot);
        g.endFill();

        g.beginFill(light, 0.9);
        g.drawCircle(0, 0, r * (0.1 + pulse * 0.05));
        g.endFill();
    }
};

// 48 — ObsidianTip: обсидиановый наконечник.
SHOWCASE_DESIGNS[48] = {
    name: "ObsidianTip",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -80);
        const light = shade(col, 50);
        const pulse = 0.5 + 0.5 * Math.sin(t * 2.5);

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(dark, 0.35);
        g.moveTo(0, -r);
        g.lineTo(r * 0.62, r * 0.55);
        g.lineTo(0, r * 0.28);
        g.lineTo(-r * 0.62, r * 0.55);
        g.closePath();
        g.endFill();

        g.lineStyle(2, light, 0.85);
        g.beginFill(col, 0.9);
        g.moveTo(0, -r * 0.72);
        g.lineTo(r * 0.4, r * 0.38);
        g.lineTo(0, r * 0.17);
        g.lineTo(-r * 0.4, r * 0.38);
        g.closePath();
        g.endFill();

        g.beginFill(0x000000, 0.42);
        g.drawCircle(0, r * 0.05, r * 0.25);
        g.endFill();

        g.beginFill(light, 0.9);
        g.drawCircle(0, r * 0.05, r * (0.09 + pulse * 0.05));
        g.endFill();
    }
};

// 49 — FinalApex: финальный тройной вершинный узел.
SHOWCASE_DESIGNS[49] = {
    name: "FinalApex",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -80);
        const light = shade(col, 75);
        const pulse = 0.5 + 0.5 * Math.sin(t * 2.6);
        const rot = t * 0.48;

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(dark, 0.3);
        starPath(g, 12, r * (0.98 + pulse * 0.08), r * 0.55, rot);
        g.endFill();

        g.lineStyle(3, light, 0.85);
        g.beginFill(col, 0.97);
        starPath(g, 3, r * 0.72, r * 0.36, rot + Math.PI / 2);
        g.endFill();

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(dark, 0.9);
        starPath(g, 3, r * (0.42 + pulse * 0.06), r * 0.2, -rot + Math.PI / 2);
        g.endFill();

        g.beginFill(light, 0.95);
        g.drawCircle(0, 0, r * (0.1 + pulse * 0.075));
        g.endFill();

        g.beginFill(0xffffff, 0.58);
        g.drawCircle(-r * 0.06, -r * 0.08, r * 0.065);
        g.endFill();

        g.lineStyle(2, light, 0.65);
        arcPath(g, r * 1.08, rot, rot + Math.PI * 0.7, 12);
        arcPath(g, r * 1.08, rot + Math.PI, rot + Math.PI * 1.7, 12);
    }
};

// 50 — Triarch2: вариант Triarch — три орбитальных треугольника.
SHOWCASE_DESIGNS[50] = {
    name: "Triarch2",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -65);
        const light = shade(col, 58);
        const rot = t * 0.7;

        g.lineStyle(3, 0x000000, 0.55);
        for (let i = 0; i < 3; i++) {
            const a = rot + i * Math.PI * 2 / 3;
            const x = Math.cos(a) * r * 0.42;
            const y = Math.sin(a) * r * 0.42;

            g.beginFill(col, 0.9);
            g.moveTo(x, y - r * 0.33);
            g.lineTo(x + r * 0.27, y + r * 0.18);
            g.lineTo(x - r * 0.27, y + r * 0.18);
            g.closePath();
            g.endFill();

            g.beginFill(light, 0.8);
            g.drawCircle(x, y, r * 0.08);
            g.endFill();
        }

        g.beginFill(dark, 0.3);
        starPath(g, 3, r * 0.72, r * 0.38, -rot);
        g.endFill();

        g.beginFill(light, 1);
        g.drawCircle(0, 0, r * 0.2);
        g.endFill();

        g.beginFill(0xffffff, 0.5);
        g.drawCircle(-r * 0.06, -r * 0.07, r * 0.06);
        g.endFill();
    }
};

// 51 — SovereignStar: королевская пятиконечная звезда.
SHOWCASE_DESIGNS[51] = {
    name: "SovereignStar",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -74);
        const light = shade(col, 68);
        const rot = -t * 0.5;

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(dark, 0.22);
        starPath(g, 5, r * 0.9, r * 0.42, rot);
        g.endFill();

        g.beginFill(col, 1);
        starPath(g, 5, r * 0.65, r * 0.29, rot + 0.16);
        g.endFill();

        g.beginFill(light, 1);
        starPath(g, 3, r * 0.31, r * 0.14, -rot);
        g.endFill();

        g.lineStyle(2, light, 0.95);
        g.drawCircle(0, 0, r * 0.48);
        g.lineStyle(0);

        g.beginFill(0xffffff, 0.46);
        g.drawCircle(-r * 0.08, -r * 0.1, r * 0.07);
        g.endFill();
    }
};

// 52 — DreadPrince: грозный ромб-принц.
SHOWCASE_DESIGNS[52] = {
    name: "DreadPrince",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -80);
        const light = shade(col, 52);
        const pulse = 0.5 + 0.5 * Math.sin(t * 2.1);

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(dark, 1);
        g.moveTo(0, -r);
        g.lineTo(r * 0.42, -r * 0.16);
        g.lineTo(r * 0.92, r * 0.58);
        g.lineTo(0, r * 0.28);
        g.lineTo(-r * 0.92, r * 0.58);
        g.lineTo(-r * 0.42, -r * 0.16);
        g.closePath();
        g.endFill();

        g.beginFill(0x000000, 0.16);
        g.drawCircle(0, r * 0.06, r * 0.45);
        g.endFill();

        g.lineStyle(2.5, light, 0.95);
        g.moveTo(0, -r * 0.8);
        g.lineTo(r * 0.58, r * 0.4);
        g.lineTo(0, r * 0.15);
        g.lineTo(-r * 0.58, r * 0.4);
        g.closePath();

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(light, 1);
        g.drawCircle(0, r * 0.05, r * (0.16 + pulse * 0.05));
        g.endFill();
    }
};

// 53 — GoldenTyrant: золотой тиран.
SHOWCASE_DESIGNS[53] = {
    name: "GoldenTyrant",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -78);
        const light = shade(col, 88);
        const pulse = 0.5 + 0.5 * Math.sin(t * 1.2);

        g.beginFill(light, 0.1);
        g.drawCircle(0, 0, r * (1.15 + pulse * 0.12));
        g.endFill();

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(dark, 0.3);
        starPath(g, 5, r * 0.9, r * 0.7, -Math.PI / 2 + t * 0.18);
        g.endFill();

        g.beginFill(col, 1);
        starPath(g, 5, r * 0.66, r * 0.28, -Math.PI / 2);
        g.endFill();

        g.lineStyle(2.5, light, 0.95);
        starPath(g, 5, r * 0.48, r * 0.2, Math.PI / 2 - t * 0.4);

        g.beginFill(light, 1);
        g.drawCircle(0, 0, r * (0.12 + pulse * 0.08));
        g.endFill();
    }
};

// 54 — RedGear: красная шестерня.
SHOWCASE_DESIGNS[54] = {
    name: "RedGear",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -65);
        const light = shade(col, 60);
        const spin = t * 0.55;

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(dark, 0.20);
        g.drawCircle(0, 0, r * 0.76);
        g.endFill();

        g.lineStyle(2, light, 0.85);
        for (let i = 0; i < 8; i++) {
            const a = spin + i * Math.PI / 4;
            const x = Math.cos(a) * r * 0.69;
            const y = Math.sin(a) * r * 0.69;
            g.beginFill(col, 1);
            g.drawRect(x - r * 0.11, y - r * 0.11, r * 0.22, r * 0.22);
            g.endFill();
        }

        g.beginFill(col, 1);
        g.drawCircle(0, 0, r * 0.58);
        g.endFill();

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(dark, 0.55);
        g.drawCircle(0, 0, r * 0.34);
        g.endFill();

        g.lineStyle(2, light, 0.9);
        g.drawRect(-r * 0.18, -r * 0.18, r * 0.36, r * 0.36);

        g.beginFill(0xffffff, 0.45);
        g.drawCircle(-r * 0.19, -r * 0.22, r * 0.09);
        g.endFill();
    }
};

// 55 — IronCrown: железная корона.
SHOWCASE_DESIGNS[55] = {
    name: "IronCrown",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -70);
        const light = shade(col, 62);
        const pulse = 0.5 + 0.5 * Math.sin(t * 1.8);

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(dark, 0.20);
        g.drawRect(-r * 0.74, -r * 0.72, r * 1.48, r * 1.44);
        g.endFill();

        g.beginFill(col, 1);
        g.moveTo(-r * 0.56, r * 0.46);
        g.lineTo(-r * 0.48, -r * 0.42);
        g.lineTo(-r * 0.18, -r * 0.18);
        g.lineTo(0, -r * 0.56);
        g.lineTo(r * 0.18, -r * 0.18);
        g.lineTo(r * 0.48, -r * 0.42);
        g.lineTo(r * 0.56, r * 0.46);
        g.closePath();
        g.endFill();

        g.lineStyle(3, dark, 0.85);
        g.moveTo(-r * 0.56, r * 0.46);
        g.lineTo(r * 0.56, r * 0.46);

        g.lineStyle(2.5, light, 0.95);
        g.drawRect(-r * 0.30, r * 0.12, r * 0.60, r * 0.27);

        g.beginFill(dark, 0.82);
        g.drawRect(-r * 0.16, -r * 0.05, r * 0.32, r * 0.32);
        g.endFill();

        g.lineStyle(2, light, 0.9);
        g.drawCircle(0, r * 0.10, r * (0.10 + pulse * 0.06));

        g.beginFill(0xffffff, 0.50);
        g.drawCircle(-r * 0.08, -r * 0.10, r * 0.055);
        g.endFill();
    }
};

// 56 — CoreVault: бронированный сейф-ядро.
SHOWCASE_DESIGNS[56] = {
    name: "CoreVault",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -80);
        const light = shade(col, 68);
        const pulse = 0.5 + 0.5 * Math.sin(t * 2.2);
        const rot = t * 0.38;

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(dark, 0.24);
        g.drawRect(-r * 0.76, -r * 0.76, r * 1.52, r * 1.52);
        g.endFill();

        g.lineStyle(4, dark, 0.95);
        g.drawRect(-r * 0.62, -r * 0.62, r * 1.24, r * 1.24);

        g.lineStyle(2.5, light, 0.95);
        g.moveTo(-r * 0.46, -r * 0.46);
        g.lineTo(r * 0.46, -r * 0.46);
        g.lineTo(r * 0.46, r * 0.46);
        g.lineTo(-r * 0.46, r * 0.46);
        g.closePath();

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(col, 1);
        g.drawRect(-r * 0.30, -r * 0.30, r * 0.60, r * 0.60);
        g.endFill();

        g.lineStyle(2, light, 0.9);
        g.drawRect(-r * 0.19, -r * 0.19, r * 0.38, r * 0.38);

        g.beginFill(0x000000, 0.26);
        g.drawCircle(0, 0, r * (0.16 + pulse * 0.07));
        g.endFill();

        g.beginFill(light, 0.90);
        g.drawCircle(0, 0, r * (0.09 + pulse * 0.05));
        g.endFill();

        for (let i = 0; i < 4; i++) {
            const a = rot + i * Math.PI / 2;
            const x = Math.cos(a) * r * 0.66;
            const y = Math.sin(a) * r * 0.66;

            g.lineStyle(2.5, light, 0.9);
            g.beginFill(col, 1);
            g.drawRect(x - r * 0.09, y - r * 0.09, r * 0.18, r * 0.18);
            g.endFill();
        }

        g.beginFill(0xffffff, 0.58);
        g.drawCircle(-r * 0.08, -r * 0.11, r * 0.05);
        g.endFill();
    }
};

// 57 — BlockBurst: кубический взрыв.
SHOWCASE_DESIGNS[57] = {
    name: "BlockBurst",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -70);
        const light = shade(col, 65);
        const pulse = 0.5 + 0.5 * Math.sin(t * 2.8);

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(dark, 0.18);
        g.drawCircle(0, 0, r * (0.66 + pulse * 0.12));
        g.endFill();

        g.lineStyle(2.5, light, 0.85);
        g.drawCircle(0, 0, r * (0.42 + pulse * 0.10));
        g.drawCircle(0, 0, r * (0.28 + pulse * 0.06));

        for (let i = 0; i < 8; i++) {
            const a = i * Math.PI / 4;
            const dist = r * (0.50 + pulse * 0.12);
            const s = r * (0.15 + pulse * 0.03);
            const x = Math.cos(a) * dist;
            const y = Math.sin(a) * dist;

            g.lineStyle(2, 0x000000, 0.55);
            g.beginFill(col, 1);
            g.drawRect(x - s * 0.5, y - s * 0.5, s, s);
            g.endFill();
        }

        g.lineStyle(3, light, 0.9);
        g.beginFill(col, 1);
        g.drawRect(-r * 0.24, -r * 0.24, r * 0.48, r * 0.48);
        g.endFill();

        g.beginFill(0xffffff, 0.48);
        g.drawCircle(-r * 0.09, -r * 0.10, r * 0.055);
        g.endFill();
    }
};

// 58 — DarkCube: тёмный куб с ромбом.
SHOWCASE_DESIGNS[58] = {
    name: "DarkCube",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -88);
        const light = shade(col, 55);
        const p = 0.5 + 0.5 * Math.sin(t * 1.8);

        g.beginFill(0x000000, 0.24);
        g.drawRect(-r * 0.70, -r * 0.62, r * 1.40, r * 1.28);
        g.endFill();

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(dark, 1);
        g.drawPolygon([
            -r * 0.60, -r * 0.60,
            r * 0.60, -r * 0.60,
            r * 0.60, r * 0.60,
            -r * 0.60, r * 0.60
        ]);
        g.endFill();

        g.beginFill(col, 0.85);
        g.drawPolygon([
            -r * 0.34, -r * 0.50,
            r * 0.50, -r * 0.34,
            r * 0.34, r * 0.50,
            -r * 0.50, r * 0.34
        ]);
        g.endFill();

        g.lineStyle(2, light, 0.85);
        g.moveTo(-r * 0.41, 0);
        g.lineTo(r * 0.41, 0);
        g.moveTo(0, -r * 0.41);
        g.lineTo(0, r * 0.41);

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(light, 0.25);
        g.drawCircle(0, 0, r * (0.13 + p * 0.06));
        g.endFill();

        g.beginFill(0xffffff, 0.42);
        g.drawCircle(-r * 0.05, -r * 0.06, r * 0.05);
        g.endFill();
    }
};

// 59 — CrimsonNode: багровый узел.
SHOWCASE_DESIGNS[59] = {
    name: "CrimsonNode",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -78);
        const light = shade(col, 64);
        const p = 0.5 + 0.5 * Math.sin(t * 2.4);
        const rot = t * 0.45;

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(dark, 1);
        g.drawPolygon([
            0, -r * 0.70,
            r * 0.70, 0,
            0, r * 0.70,
            -r * 0.70, 0
        ]);
        g.endFill();

        for (let i = 0; i < 4; i++) {
            const a = rot + i * Math.PI / 2;
            const x = Math.cos(a) * r * 0.48;
            const y = Math.sin(a) * r * 0.48;
            g.lineStyle(2, light, 0.9);
            g.beginFill(col, 1);
            g.drawRect(x - r * 0.08, y - r * 0.08, r * 0.16, r * 0.16);
            g.endFill();
        }

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(col, 1);
        g.drawRect(-r * 0.24, -r * 0.24, r * 0.48, r * 0.48);
        g.endFill();

        g.beginFill(light, 0.25);
        g.drawCircle(0, 0, r * (0.15 + p * 0.07));
        g.endFill();
    }
};

// 60 — StarCore: пятиконечное ядро-звезда.
SHOWCASE_DESIGNS[60] = {
    name: "StarCore",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -55);
        const light = shade(col, 65);
        const pulse = 0.5 + 0.5 * Math.sin(t * 2.4);

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(dark, 0.3);
        starPath(g, 5, r, r * 0.34, t * 0.25);
        g.endFill();

        g.lineStyle(2, light, 0.9);
        starPath(g, 5, r * 0.82, r * 0.68, -t * 0.55);

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(col, 1);
        starPath(g, 5, r * 0.52, r * 0.23, t * 0.5);
        g.endFill();

        g.beginFill(light, 0.95);
        g.drawCircle(0, 0, r * (0.16 + pulse * 0.07));
        g.endFill();

        g.beginFill(0xffffff, 0.55);
        g.drawCircle(-r * 0.06, -r * 0.08, r * 0.06);
        g.endFill();
    }
};

// 61 — OrbitStar: звезда с орбитальными спутниками.
SHOWCASE_DESIGNS[61] = {
    name: "OrbitStar",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -60);
        const light = shade(col, 65);
        const pulse = 0.5 + 0.5 * Math.sin(t * 1.5);

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(dark, 0.22);
        starPath(g, 5, r * 0.64, r * 0.5, t * 0.4);
        g.endFill();

        g.lineStyle(2, light, 0.85);
        g.drawCircle(0, 0, r * 0.82);

        for (let i = 0; i < 5; i++) {
            const a = t * 0.7 + i * Math.PI * 0.4;
            g.beginFill(light, 0.9);
            g.drawCircle(Math.cos(a) * r * 0.82, Math.sin(a) * r * 0.82, r * 0.075);
            g.endFill();
        }

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(col, 1);
        starPath(g, 5, r * 0.52, r * 0.22, -t * 0.45);
        g.endFill();

        g.beginFill(light, 0.95);
        g.drawCircle(0, 0, r * (0.13 + pulse * 0.06));
        g.endFill();

        g.beginFill(0xffffff, 0.5);
        g.drawCircle(-r * 0.05, -r * 0.06, r * 0.05);
        g.endFill();
    }
};

// 62 — RoyalHex: королевский гексагон.
SHOWCASE_DESIGNS[62] = {
    name: "RoyalHex",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const light = shade(col, 55);
        const dark = shade(col, -60);

        g.beginFill(dark, 0.2);
        starPath(g, 6, r * 0.92, r * 0.72, t * 0.25);
        g.endFill();

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(col, 1);
        starPath(g, 5, r * 0.7, r * 0.52, -t * 0.4);
        g.endFill();

        g.lineStyle(2, light, 0.85);
        starPath(g, 6, r * 0.48, r * 0.4, t * 0.7);
        g.lineStyle(0);

        g.beginFill(light, 0.85);
        g.drawCircle(0, 0, r * 0.17);
        g.endFill();
    }
};

// 63 — NovaCrest: вспышка-гребень.
SHOWCASE_DESIGNS[63] = {
    name: "NovaCrest",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const light = shade(col, 70);
        const dark = shade(col, -65);
        const pulse = 0.5 + 0.5 * Math.sin(t * 1.8);

        g.beginFill(light, 0.13);
        g.drawCircle(0, 0, r * 1.1);
        g.endFill();

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(dark, 0.28);
        starPath(g, 6, r * 0.98, r * 0.72, t * 0.25);
        g.endFill();

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(col, 1);
        starPath(g, 5, r * 0.78, r * 0.42, -t * 0.35);
        g.endFill();

        g.lineStyle(2, light, 0.9);
        starPath(g, 5, r * 0.48, r * 0.25, t * 0.7);

        g.lineStyle(0);
        g.beginFill(light, 0.8);
        g.drawCircle(0, 0, r * (0.12 + pulse * 0.1));
        g.endFill();
    }
};

// 64 — Pentagonix: пятиугольник с двойным контуром.
SHOWCASE_DESIGNS[64] = {
    name: "Pentagonix",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const light = shade(col, 65);
        const dark = shade(col, -65);
        const rot = t * 0.45;

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(col, 1);
        starPath(g, 5, r * 0.78, r * 0.61, rot);
        g.endFill();

        g.lineStyle(2, light, 0.85);
        starPath(g, 5, r * 0.57, r * 0.4, -rot);

        g.lineStyle(3, dark, 0.9);
        g.drawCircle(0, 0, r * 0.28);

        g.lineStyle(0);
        g.beginFill(light, 0.45);
        g.drawCircle(0, 0, r * 0.17);
        g.endFill();

        g.beginFill(0xffffff, 0.55);
        g.drawCircle(-r * 0.055, -r * 0.07, r * 0.055);
        g.endFill();
    }
};

// 65 — StarVault: хранилище-звезда.
SHOWCASE_DESIGNS[65] = {
    name: "StarVault",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -65);
        const light = shade(col, 70);
        const rot = t * 0.32;

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(dark, 0.3);
        g.drawCircle(0, 0, r * 0.82);
        g.endFill();

        g.lineStyle(3, light, 0.8);
        starPath(g, 5, r * 0.72, r * 0.57, rot);

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(col, 1);
        g.drawCircle(0, 0, r * 0.43);
        g.endFill();

        g.lineStyle(2, light, 0.9);
        starPath(g, 5, r * 0.3, r * 0.13, -rot * 2);

        g.lineStyle(0);
        g.beginFill(light, 0.35);
        g.drawCircle(0, 0, r * 0.22);
        g.endFill();
    }
};

// 66 — CogCore: шестерня-ядро.
SHOWCASE_DESIGNS[66] = {
    name: "CogCore",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -70);
        const light = shade(col, 55);

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(dark, 0.3);
        starPath(g, 8, r * 0.9, r * 0.67, t * 0.35);
        g.endFill();

        g.beginFill(col, 1);
        starPath(g, 8, r * 0.65, r * 0.5, -t * 0.5);
        g.endFill();

        g.lineStyle(2, light, 0.9);
        g.drawCircle(0, 0, r * 0.4);

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(dark, 1);
        g.drawCircle(0, 0, r * 0.24);
        g.endFill();

        g.beginFill(light, 1);
        g.drawCircle(0, 0, r * 0.11);
        g.endFill();
    }
};

// 67 — HexaAeon: гекса-эон с орбитами.
SHOWCASE_DESIGNS[67] = {
    name: "HexaAeon",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -70);
        const light = shade(col, 70);
        const pulse = 0.5 + 0.5 * Math.sin(t * 1.5);
        const rot = t * 0.27;

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(dark, 0.22);
        starPath(g, 6, r * 0.91, r * 0.7, rot);
        g.endFill();

        g.lineStyle(2, light, 0.82);
        starPath(g, 6, r * 0.73, r * 0.61, -rot * 1.7);
        g.drawCircle(0, 0, r * 0.55);

        for (let i = 0; i < 6; i++) {
            const a = rot + i * Math.PI / 3;
            const rr = r * (0.55 + pulse * 0.08);
            g.lineStyle(3, 0x000000, 0.55);
            g.beginFill(light, 0.9);
            starPath(g, 6, r * 0.13, r * 0.085, a);
            g.endFill();
            g.beginFill(col, 1);
            g.drawCircle(Math.cos(a) * rr, Math.sin(a) * rr, r * 0.08);
            g.endFill();
        }

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(col, 1);
        starPath(g, 6, r * (0.33 + pulse * 0.025), r * 0.2, -t * 0.55);
        g.endFill();

        g.beginFill(0xffffff, 0.52);
        g.drawCircle(-r * 0.09, -r * 0.1, r * 0.075);
        g.endFill();
    }
};

// 68 — FractalBloom: фрактальный цветок.
SHOWCASE_DESIGNS[68] = {
    name: "FractalBloom",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -60);
        const light = shade(col, 55);
        const pulse = 0.5 + 0.5 * Math.sin(t * 1.5);

        g.beginFill(dark, 0.18);
        g.drawCircle(0, r * 0.12, r * 0.82);
        g.endFill();

        g.lineStyle(3, 0x000000, 0.55);
        for (let i = 0; i < 8; i++) {
            const a = i * Math.PI / 4 + t * 0.18;
            const x = Math.cos(a) * r * 0.48;
            const y = Math.sin(a) * r * 0.48;
            g.beginFill(light, 0.72);
            g.drawCircle(x, y, r * (0.27 + pulse * 0.025));
            g.endFill();
        }

        g.beginFill(col, 1);
        g.drawCircle(0, 0, r * (0.3 + pulse * 0.055));
        g.endFill();

        g.beginFill(0xffffff, 0.55);
        g.drawCircle(-r * 0.1, -r * 0.12, r * 0.085);
        g.endFill();
    }
};

// 69 — SpiralNest: спиральное гнездо.
SHOWCASE_DESIGNS[69] = {
    name: "SpiralNest",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -60);
        const light = shade(col, 60);
        const p = 0.5 + 0.5 * Math.sin(t * 1.5);

        g.beginFill(dark, 0.2);
        g.drawCircle(0, r * 0.08, r * 0.78);
        g.endFill();

        g.lineStyle(3, 0x000000, 0.55);
        for (let ring = 0; ring < 3; ring++) {
            const count = 7 + ring;
            for (let i = 0; i < count; i++) {
                const a = t * (0.18 + ring * 0.08) + i * Math.PI * 2 / count;
                const rr = r * (0.22 + ring * 0.17);
                g.beginFill(light, 0.7);
                g.drawCircle(Math.cos(a) * rr, Math.sin(a) * rr, r * 0.075);
                g.endFill();
            }
        }

        g.beginFill(col, 1);
        g.drawCircle(0, 0, r * (0.2 + p * 0.06));
        g.endFill();
    }
};

// 70 — FractureStar: звезда с трещиной-разломами.
SHOWCASE_DESIGNS[70] = {
    name: "FractureStar",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -65);
        const light = shade(col, 65);
        const pulse = 0.5 + 0.5 * Math.sin(t * 1.4);

        g.beginFill(dark, 0.2);
        starPath(g, 8, r * 0.86, r * 0.5, t * 0.15);
        g.endFill();

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(col, 0.85);
        starPath(g, 8, r * 0.65, r * 0.3, -t * 0.25);
        g.endFill();

        g.lineStyle(2, light, 0.8);
        starPath(g, 8, r * 0.43, r * 0.22, t * 0.4);

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(light, 1);
        g.drawCircle(0, 0, r * (0.2 + pulse * 0.07));
        g.endFill();
    }
};

// 71 — FractalOrb: фрактальная сфера из колец точек.
SHOWCASE_DESIGNS[71] = {
    name: "FractalOrb",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const light = shade(col, 55);
        const dark = shade(col, -65);
        const p = 0.5 + 0.5 * Math.sin(t * 1.9);

        g.lineStyle(3, 0x000000, 0.55);
        for (let level = 0; level < 3; level++) {
            const count = 6 + level * 2;
            const rr = r * (0.3 + level * 0.17);
            for (let i = 0; i < count; i++) {
                const a = i * Math.PI * 2 / count + t * (0.2 + level * 0.15);
                g.beginFill(level === 2 ? light : dark, 0.7);
                g.drawCircle(Math.cos(a) * rr, Math.sin(a) * rr, r * (0.1 - level * 0.018));
                g.endFill();
            }
        }

        g.beginFill(col, 1);
        g.drawCircle(0, 0, r * (0.22 + p * 0.07));
        g.endFill();
    }
};

// 72 — FractalGod: фрактальный бог.
SHOWCASE_DESIGNS[72] = {
    name: "FractalGod",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -75);
        const light = shade(col, 75);
        const p = 0.5 + 0.5 * Math.sin(t * 1.5);

        g.beginFill(dark, 0.2);
        g.drawCircle(0, 0, r * (0.9 + p * 0.12));
        g.endFill();

        g.lineStyle(3, 0x000000, 0.55);
        starPath(g, 12, r * 0.82, r * 0.58, t * 0.12);
        g.beginFill(dark, 0.7);
        g.endFill();

        g.lineStyle(2.5, light, 0.85);
        starPath(g, 8, r * 0.62, r * 0.38, -t * 0.32);

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(col, 1);
        g.drawCircle(0, 0, r * (0.3 + p * 0.065));
        g.endFill();

        g.lineStyle(2, light, 0.8);
        g.drawCircle(0, 0, r * (0.42 + p * 0.06));

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(0xffffff, 0.65);
        g.drawCircle(-r * 0.09, -r * 0.11, r * 0.08);
        g.endFill();

        g.beginFill(light, 0.35);
        g.drawCircle(r * 0.18, r * 0.18, r * 0.055);
        g.endFill();
    }
};

// 73 — VoidFlower: цветок пустоты.
SHOWCASE_DESIGNS[73] = {
    name: "VoidFlower",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -75);
        const light = shade(col, 65);
        const pulse = 0.5 + 0.5 * Math.sin(t * 1.3);

        g.beginFill(dark, 0.3);
        g.drawCircle(0, r * 0.16, r * 0.67);
        g.endFill();

        g.lineStyle(3, 0x000000, 0.55);
        for (let i = 0; i < 7; i++) {
            const a = t * 0.12 + i * Math.PI * 2 / 7;
            const x = Math.cos(a) * r * 0.35;
            const y = Math.sin(a) * r * 0.35;
            g.beginFill(shade(col, -10), 0.86);
            starPath(g, 5, r * 0.46, r * 0.19, a);
            g.endFill();
        }

        g.lineStyle(2, light, 0.9);
        starPath(g, 14, r * 0.63, r * 0.52, -t * 0.42);

        g.lineStyle(0);
        g.beginFill(0x000000, 0.75);
        g.drawCircle(0, 0, r * (0.27 + pulse * 0.045));
        g.endFill();

        g.beginFill(light, 0.85);
        g.drawCircle(0, 0, r * 0.12);
        g.endFill();

        g.beginFill(0xffffff, 0.5);
        g.drawCircle(-r * 0.035, -r * 0.04, r * 0.042);
        g.endFill();
    }
};

// 74 — FractShell: фрактальная раковина.
SHOWCASE_DESIGNS[74] = {
    name: "FractShell",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -70);
        const light = shade(col, 60);
        const pulse = 0.5 + 0.5 * Math.sin(t * 1.8);

        g.beginFill(dark, 0.25);
        g.drawCircle(0, r * 0.15, r * 0.68);
        g.endFill();

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(col, 0.88);
        starPath(g, 14, r * 0.7, r * 0.58, -t * 0.14);
        g.endFill();

        for (let layer = 0; layer < 4; layer++) {
            const rr = r * (0.2 + layer * 0.13 + pulse * 0.015);
            g.lineStyle(2, light, 0.82);
            starPath(g, 7 + layer, rr, rr * 0.84, t * (0.2 + layer * 0.05));
        }

        g.lineStyle(2, 0x000000, 0.45);
        for (let i = 0; i < 7; i++) {
            const a = t * 0.27 + i * Math.PI * 2 / 7;
            const rr = r * 0.58;
            g.beginFill(light, 0.75);
            g.drawCircle(Math.cos(a) * rr, Math.sin(a) * rr, r * 0.05);
            g.endFill();
        }

        g.lineStyle(0);
        g.beginFill(col, 1);
        g.drawCircle(0, 0, r * 0.16);
        g.endFill();
    }
};

// 75 — Moon: луна с кратерами.
SHOWCASE_DESIGNS[75] = {
    name: "Moon",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -70);
        const light = shade(col, 45);
        const drift = Math.sin(t * 0.7) * r * 0.04;

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(dark, 0.3);
        g.drawCircle(2, 4, r * 0.72);
        g.endFill();

        g.beginFill(col, 1);
        g.drawCircle(drift, 0, r * 0.68);
        g.endFill();

        g.beginFill(light, 0.22);
        g.drawCircle(-r * 0.12, -r * 0.05, r * 0.59);
        g.endFill();

        g.beginFill(0x000000, 0.25);
        g.drawCircle(-r * 0.23, -r * 0.08, r * 0.1);
        g.drawCircle(r * 0.2, r * 0.12, r * 0.08);
        g.drawCircle(r * 0.04, -r * 0.25, r * 0.055);
        g.endFill();

        g.lineStyle(2, light, 0.7);
        g.drawCircle(0, 0, r * 0.68);
    }
};

// 76 — Exoplanet: экзопланета с кольцами.
SHOWCASE_DESIGNS[76] = {
    name: "Exoplanet",
    draw: function (g, r, col, t) {
        g.lineStyle(0);

        const planet = 0x6d4f91;
        const hot = 0xc75e64;
        const dark = 0x352b40;

        for (let i = 0; i < 3; i++) {
            const rr = r * (0.7 + i * 0.13);
            g.lineStyle(2, i === 1 ? 0xe74c3c : 0x9b59b6, 0.4);

            const pts = [];
            for (let j = 0; j <= 24; j++) {
                const a = j / 24 * Math.PI * 2 + t * (0.2 + i * 0.12);
                pts.push(Math.cos(a) * rr, Math.sin(a) * rr * 0.55);
            }
            g.drawPolygon(pts);

            g.beginFill(0xff6b6b, 0.55);
            const a = t * (0.5 + i * 0.2) + i;
            g.drawCircle(
                Math.cos(a) * rr,
                Math.sin(a) * rr * 0.55,
                r * 0.05
            );
            g.endFill();
        }

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(dark, 1);
        g.drawCircle(0, 0, r * 0.45);
        g.endFill();

        g.beginFill(planet, 1);
        g.drawCircle(-r * 0.04, -r * 0.03, r * 0.4);
        g.endFill();

        g.beginFill(hot, 0.65);
        g.drawCircle(r * 0.13, r * 0.04, r * 0.14);
        g.endFill();

        g.beginFill(0xffffff, 0.45);
        g.drawCircle(-r * 0.13, -r * 0.16, r * 0.07);
        g.endFill();
    }
};

// 77 — PointNova: новичковый пульсар.
SHOWCASE_DESIGNS[77] = {
    name: "PointNova",
    draw: function (g, r, col, t) {
        g.lineStyle(0);
        const dark = shade(col, -65);
        const light = shade(col, 70);
        const p = 0.5 + 0.5 * Math.sin(t * 1.7);
        const a = t * 0.32;

        g.beginFill(dark, 0.2);
        g.drawCircle(0, r * 0.18, r * 0.82);
        g.endFill();

        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(col, 1);
        g.drawCircle(0, 0, r * 0.5);
        g.endFill();

        g.beginFill(light, 0.25 + p * 0.15);
        g.drawCircle(0, 0, r * (0.63 + p * 0.1));
        g.endFill();

        for (let i = 0; i < 8; i++) {
            const q = a + i * Math.PI / 4;
            const rr = r * (0.69 + p * 0.08);
            g.beginFill(light, 0.95);
            g.drawCircle(Math.cos(q) * rr, Math.sin(q) * rr, r * 0.075);
            g.endFill();
        }

        g.beginFill(light, 1);
        g.drawCircle(0, 0, r * (0.16 + p * 0.04));
        g.endFill();

        g.beginFill(0xffffff, 0.7);
        g.drawCircle(-r * 0.17, -r * 0.2, r * 0.09);
        g.endFill();
    }
};