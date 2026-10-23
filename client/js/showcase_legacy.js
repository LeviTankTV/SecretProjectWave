// showcase_legacy.js - legacy-дизайны витрины TestRoom (idx 1-23), вынесены из drawShowcaseDesign.

function drawShowcaseLegacy(g, idx, size, col) {
    g.lineStyle(0);
    const r = size;
    const t = performance.now() / 1000;
    const dark = shade(col, -60);
    const light = shade(col, 55);

    if (idx === 1) {
        // Spike: шар с 8 треугольными шипами и светлым ядром (зона Triangle).
        g.beginFill(dark, 0.22);
        g.drawCircle(0, 0, r * 1.35);
        g.endFill();
        g.lineStyle(3, 0x000000, 0.5);
        g.beginFill(col, 1);
        g.drawCircle(0, 0, r * 0.72);
        g.endFill();
        g.beginFill(dark, 0.25);
        g.drawCircle(0, -r * 0.12, r * 0.72);
        g.endFill();
        for (let i = 0; i < 8; i++) {
            const a = i * Math.PI / 4 + Math.PI / 8;
            const cx = Math.cos(a) * r * 0.72;
            const cy = Math.sin(a) * r * 0.72;
            g.beginFill(light, 1);
            g.moveTo(cx, cy);
            g.lineTo(Math.cos(a) * r * 1.25 - Math.sin(a) * r * 0.18, Math.sin(a) * r * 1.25 + Math.cos(a) * r * 0.18);
            g.lineTo(Math.cos(a) * r * 0.72 - Math.sin(a) * r * 0.42, Math.sin(a) * r * 0.72 + Math.cos(a) * r * 0.42);
            g.closePath();
            g.endFill();
        }
        g.lineStyle(2.5, light, 0.95);
        g.beginFill(col, 1);
        g.drawCircle(0, 0, r * 0.34);
        g.endFill();
        g.beginFill(0xffffff, 0.95);
        g.drawCircle(0, 0, r * 0.11);
        g.endFill();
    } else if (idx === 2) {
        // Star: пятиконечная звезда с двойной обводкой и бликом (зона Polygon).
        g.beginFill(light, 0.2);
        g.drawCircle(0, 0, r * 1.4);
        g.endFill();
        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(col, 1);
        starPath(g, 5, r, r * 0.48, -Math.PI / 2);
        g.endFill();
        g.beginFill(dark, 0.3);
        starPath(g, 5, r * 0.72, r * 0.36, -Math.PI / 2);
        g.endFill();
        g.lineStyle(2.5, light, 0.9);
        g.beginFill(col, 1);
        g.drawCircle(0, 0, r * 0.4);
        g.endFill();
        g.lineStyle(2, 0xffffff, 0.95);
        g.beginFill(light, 1);
        g.drawCircle(0, 0, r * 0.16);
        g.endFill();
    } else if (idx === 3) {
        // Beads: «бусы» — два шарика-моба, соединённые гибкой шейкой. Голова
        // (агрит головой) смотрит на +x, второй шар болтается сзади волной.
        // Зона Point.
        g.beginFill(dark, 0.16);
        g.drawEllipse(-r * 0.1, 0, r * 1.5, r * 0.85);
        g.endFill();
        const sway = Math.sin(t * 2.4) * r * 0.14;
        // Задняя бусина (второй моб) сзади, покачивается.
        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(light, 1);
        g.drawCircle(-r * 0.95, sway, r * 0.42);
        g.endFill();
        g.lineStyle(0);
        g.beginFill(dark, 0.25);
        g.drawCircle(-r * 0.95, sway + r * 0.12, r * 0.38);
        g.endFill();
        g.beginFill(0xffffff, 0.5);
        g.drawCircle(-r * 1.08, sway - r * 0.12, r * 0.1);
        g.endFill();
        // Шейка-связка между двумя бусинами.
        g.lineStyle(2.5, 0x000000, 0.5);
        g.beginFill(col, 1);
        g.drawCircle(-r * 0.15, sway * 0.5, r * 0.26);
        g.endFill();
        g.lineStyle(0);
        // Головная бусина.
        g.lineStyle(3, 0x000000, 0.6);
        g.beginFill(col, 1);
        g.drawCircle(r * 0.55, 0, r * 0.6);
        g.endFill();
        g.lineStyle(0);
        g.beginFill(dark, 0.22);
        g.drawCircle(r * 0.55, r * 0.2, r * 0.57);
        g.endFill();
        // Антенны.
        g.lineStyle(2.5, light, 0.95);
        g.moveTo(r * 0.7, -r * 0.48); g.lineTo(r * 1.05, -r * 0.95);
        g.moveTo(r * 0.4, -r * 0.55); g.lineTo(r * 0.45, -r * 1.08);
        g.lineStyle(0);
        g.beginFill(light, 1);
        g.drawCircle(r * 1.05, -r * 0.95, r * 0.06);
        g.drawCircle(r * 0.45, -r * 1.08, r * 0.06);
        g.endFill();
        // Улыбка.
        g.lineStyle(2, 0x000000, 0.5);
        g.arc(r * 0.45, r * 0.28, r * 0.24, 0.25, Math.PI - 0.55);
        g.lineStyle(0);
    } else if (idx === 4) {
        // Tear: капля с бликом и внутренним градиентом (зона Point).
        g.beginFill(light, 0.18);
        g.drawCircle(0, -r * 0.1, r * 1.4);
        g.endFill();
        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(col, 1);
        g.moveTo(0, -r);
        g.bezierCurveTo(r * 0.95, -r * 0.2, r * 0.85, r * 0.7, 0, r);
        g.bezierCurveTo(-r * 0.85, r * 0.7, -r * 0.95, -r * 0.2, 0, -r);
        g.closePath();
        g.endFill();
        g.beginFill(dark, 0.3);
        g.moveTo(0, -r * 0.35);
        g.bezierCurveTo(r * 0.8, -r * 0.05, r * 0.7, r * 0.6, 0, r);
        g.closePath();
        g.endFill();
        g.lineStyle(3, 0xffffff, 0.9);
        g.beginFill(light, 1);
        g.drawEllipse(-r * 0.3, -r * 0.35, r * 0.14, r * 0.24);
        g.endFill();
        g.beginFill(0xffffff, 1);
        g.drawCircle(-r * 0.28, -r * 0.4, r * 0.07);
        g.endFill();
    } else if (idx === 5) {
        // Cross: квадратный гем-медальон ПЛЮС для Square-зоны — острые
        // прямые ветки (без скруглений), ромбовая оправа-«оправа» позади и
        // вращающееся ромбовое ядро; квадратные «заклёпки» на концах веток.
        const rot5 = t * 0.9;
        // Ромб-оправа позади корпуса (лёгкий пульс + вращение).
        const w5 = r * 1.26 * (1 + 0.03 * Math.sin(t * 2.4));
        const diaPts = (rad, base) => {
            for (let i = 0; i < 4; i++) {
                const a5 = base + (i / 4) * Math.PI * 2;
                const x = Math.cos(a5) * rad, y = Math.sin(a5) * rad;
                if (i === 0) g.moveTo(x, y);
                else g.lineTo(x, y);
            }
            g.closePath();
        };
        g.lineStyle(2.4, dark, 0.95);
        diaPts(w5, rot5 * 0.35);
        g.lineStyle(1.2, light, 0.45);
        diaPts(w5 * 0.82, rot5 * 0.35);
        // Острый плюс-корпус: два прямоугольника без скруглений.
        g.lineStyle(3, 0x000000, 0.6);
        g.beginFill(col, 1);
        g.drawRect(-r * 0.3, -r, r * 0.6, r * 2);
        g.drawRect(-r, -r * 0.3, r * 2, r * 0.6);
        g.endFill();
        g.lineStyle(0);
        // Фасетные тени нижней и правой ветки (свет сверху-слева).
        g.beginFill(dark, 0.28);
        g.drawRect(-r * 0.3, r * 0.34, r * 0.6, r * 0.66);
        g.drawRect(r * 0.34, -r * 0.3, r * 0.66, r * 0.6);
        g.endFill();
        // Светлая внутренняя кромка по контуру плюса (гранёность).
        g.lineStyle(1.4, light, 0.55);
        g.drawRect(-r * 0.18, -r * 0.86, r * 0.36, r * 1.72);
        g.drawRect(-r * 0.86, -r * 0.18, r * 1.72, r * 0.36);
        // Вращающееся ромбовое ядро по центру.
        g.lineStyle(2, 0xffffff, 0.9);
        g.beginFill(col, 1);
        diaPts(r * 0.5, Math.PI / 4 + rot5);
        g.endFill();
        g.lineStyle(0);
        // Фасет тени: нижний сектор повернутого ромба (от боковой вершины
        // через нижнюю к центру).
        const b5 = Math.PI / 4 + rot5;
        g.beginFill(dark, 0.3);
        g.moveTo(Math.cos(b5) * r * 0.5, Math.sin(b5) * r * 0.5);
        g.lineTo(Math.cos(b5 + Math.PI / 2) * r * 0.5, Math.sin(b5 + Math.PI / 2) * r * 0.5);
        g.lineTo(0, 0);
        g.closePath();
        g.endFill();
        g.lineStyle(1.2, light, 0.8);
        diaPts(r * 0.26, Math.PI / 4 + rot5);
        g.lineStyle(0);
        // Квадратные «заклёпки» на концах веток (пульс).
        const g5 = 0.5 + 0.5 * Math.sin(t * 3);
        g.beginFill(light, 0.55 + 0.35 * g5);
        for (const [dx, dy] of [[0, -r * 0.88], [0, r * 0.88], [-r * 0.88, 0], [r * 0.88, 0]]) {
            g.drawRect(dx - r * 0.09, dy - r * 0.09, r * 0.18, r * 0.18);
        }
        g.endFill();
    } else if (idx === 6) {
        // Saturn: планета с наклонным кольцом, проходящим ЗА планетой (верхняя
        // дуга) и ПЕРЕД ней (нижняя дуга), с яркими полосами и луной на орбите.
        // Никаких прозрачных обводок поверх планеты.
        g.beginFill(0x0a0f15, 0.45);
        g.drawCircle(r * 0.06, r * 0.3, r * 1.02);
        g.endFill();
        g.beginFill(light, 0.1);
        g.drawCircle(0, 0, r * 1.3);
        g.endFill();
        const tilt = -0.38;
        const rx = r * 1.32, ry = r * 0.4;
        const ept = (a, k) => {
            const x = Math.cos(a) * rx * k, y = Math.sin(a) * ry * k;
            return [x * Math.cos(tilt) - y * Math.sin(tilt), x * Math.sin(tilt) + y * Math.cos(tilt)];
        };
        const bandArc = (from, to, w, cc, aa, k) => {
            g.lineStyle(w, cc, aa);
            for (let i = 0; i <= 26; i++) {
                const a = from + (to - from) * i / 26;
                const p = ept(a, k);
                if (i === 0) g.moveTo(p[0], p[1]);
                else g.lineTo(p[0], p[1]);
            }
        };
        // ЗАДНЯЯ дуга (за планетой): верхняя половина эллипса.
        bandArc(Math.PI, 2 * Math.PI, 8, dark, 0.85, 1);
        bandArc(Math.PI, 2 * Math.PI, 4, light, 0.9, 0.98);
        bandArc(Math.PI, 2 * Math.PI, 1.6, 0xffffff, 0.7, 0.94);
        g.lineStyle(0);
        // Планета: газовый гигант с полосами и терминатором.
        g.lineStyle(3, 0x000000, 0.6);
        g.beginFill(col, 1);
        g.drawCircle(0, 0, r * 0.66);
        g.endFill();
        g.lineStyle(0);
        g.beginFill(dark, 0.4);
        g.drawEllipse(0, r * 0.14, r * 0.66, r * 0.52);
        g.endFill();
        g.beginFill(light, 0.3);
        g.drawEllipse(-r * 0.08, -r * 0.3, r * 0.52, r * 0.24);
        g.endFill();
        g.beginFill(0xffffff, 0.5);
        g.drawCircle(-r * 0.24, -r * 0.3, r * 0.13);
        g.endFill();
        g.beginFill(dark, 0.35);
        g.drawEllipse(r * 0.3, r * 0.05, r * 0.18, r * 0.1);
        g.endFill();
        // ПЕРЕДНЯЯ дуга (перед планетой): нижняя половина эллипса.
        bandArc(0, Math.PI, 8, dark, 0.9, 1);
        bandArc(0, Math.PI, 4, light, 1, 0.98);
        bandArc(0, Math.PI, 1.6, 0xffffff, 0.8, 0.94);
        g.lineStyle(0);
        // Луна на орбите.
        g.beginFill(0xdfe8f2, 1);
        g.drawCircle(r * 1.02, -r * 0.92, r * 0.1);
        g.endFill();
        g.lineStyle(1.2, 0xffffff, 0.55);
        g.drawCircle(r * 1.02, -r * 0.92, r * 0.16);
        g.lineStyle(0);
    } else if (idx === 7) {
        // Sun: солнце с 12 лучами, пульсирующей короной и ядром (зона Fractal).
        const pulse = 1 + Math.sin(t * 2.2) * 0.05;
        g.beginFill(light, 0.2);
        g.drawCircle(0, 0, r * 1.5 * pulse);
        g.endFill();
        g.lineStyle(3, 0x000000, 0.4);
        g.beginFill(col, 1);
        starPath(g, 12, r * pulse, r * 0.82, 0);
        g.endFill();
        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(col, 1);
        g.drawCircle(0, 0, r * 0.7);
        g.endFill();
        g.beginFill(dark, 0.25);
        g.drawCircle(0, r * 0.08, r * 0.7);
        g.endFill();
        g.lineStyle(2.5, 0xffffff, 0.95);
        g.beginFill(light, 1);
        g.drawCircle(0, 0, r * 0.3);
        g.endFill();
        g.beginFill(0xffffff, 1);
        g.drawCircle(0, 0, r * 0.1);
        g.endFill();
    } else if (idx === 8) {
        // Eye: «глаз» — миндалина, радужка и зрачок с бликом (зона Chaos).
        g.beginFill(dark, 0.22);
        g.drawCircle(0, 0, r * 1.4);
        g.endFill();
        g.lineStyle(3, 0x000000, 0.6);
        g.beginFill(col, 1);
        g.drawEllipse(0, 0, r, r * 0.78);
        g.endFill();
        g.lineStyle(2.5, 0x000000, 0.5);
        g.beginFill(dark, 0.5);
        g.drawEllipse(0, 0, r * 0.82, r * 0.62);
        g.endFill();
        g.beginFill(light, 1);
        g.drawEllipse(0, 0, r * 0.6, r * 0.46);
        g.endFill();
        g.beginFill(0x000000, 0.9);
        g.drawCircle(0, 0, r * 0.22);
        g.endFill();
        g.beginFill(0xffffff, 1);
        g.drawCircle(-r * 0.18, -r * 0.15, r * 0.09);
        g.endFill();
        g.lineStyle(2, light, 0.9);
        g.beginFill(dark, 0.3);
        g.drawEllipse(0, -r * 0.55, r * 0.55, r * 0.12);
        g.endFill();
        g.lineStyle(0);
    } else if (idx === 9) {
        // Galaxy: живая спиральная галактика — два вращающихся рукава из
        // мерцающих звёздных точек, пульсирующее ядро и орбитальная пыль.
        g.beginFill(0x0a0618, 0.55);
        g.drawCircle(0, 0, r * 1.35);
        g.endFill();
        const arms = 2, steps = 34;
        for (let arm = 0; arm < arms; arm++) {
            for (let i = 0; i < steps; i++) {
                const f = i / steps;
                const ang = t * 0.9 + arm * Math.PI + f * 3.6;
                const rad = f * r * 1.15;
                const x = Math.cos(ang) * rad;
                const y = Math.sin(ang) * rad * 0.62;
                const sz = (1 - f) * r * 0.085 + r * 0.03;
                const tw = 0.55 + 0.45 * Math.sin(t * 3 + i * 0.7 + arm * 2);
                g.beginFill(i % 3 === 0 ? 0xffd9f2 : light, tw);
                g.drawCircle(x, y, sz);
                g.endFill();
            }
        }
        g.beginFill(0xffffff, 0.7);
        for (let i = 0; i < 10; i++) {
            const a = i * 2.399 + t * 0.25;
            const rad = r * (0.5 + 0.75 * ((i * 37) % 10) / 10);
            g.drawCircle(Math.cos(a) * rad, Math.sin(a) * rad * 0.72, r * 0.035);
        }
        g.endFill();
        const np = 1 + Math.sin(t * 2.6) * 0.12;
        g.beginFill(light, 0.35);
        g.drawEllipse(0, 0, r * 0.55 * np, r * 0.4 * np);
        g.endFill();
        g.beginFill(0xffe9fb, 0.95);
        g.drawEllipse(0, 0, r * 0.3 * np, r * 0.22 * np);
        g.endFill();
        g.beginFill(0xffffff, 1);
        g.drawCircle(0, 0, r * 0.1 * np);
        g.endFill();
    } else if (idx === 10) {
        // Mote: тёмное ядро с орбитальными «мотыльками»-искрами (Fractal).
        g.beginFill(dark, 0.25);
        g.drawCircle(0, 0, r * 1.4);
        g.endFill();
        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(col, 1);
        g.drawCircle(0, 0, r * 0.55);
        g.endFill();
        g.lineStyle(0);
        g.beginFill(dark, 0.4);
        g.drawCircle(0, r * 0.08, r * 0.55);
        g.endFill();
        g.lineStyle(2, light, 0.9);
        g.beginFill(light, 0.8);
        g.drawCircle(0, 0, r * 0.22);
        g.endFill();
        g.lineStyle(0);
        g.lineStyle(1.2, light, 0.4);
        g.drawEllipse(0, 0, r * 1.1, r * 0.45);
        g.drawEllipse(0, 0, r * 0.45, r * 1.1);
        g.lineStyle(0);
        g.beginFill(0xffffff, 1);
        const m1 = t * 1.7, m2 = -t * 1.2 + 1;
        g.drawCircle(Math.cos(m1) * r * 1.1, Math.sin(m1) * r * 0.45, r * 0.07);
        g.drawCircle(Math.cos(m2) * r * 0.45, Math.sin(m2) * r * 1.1, r * 0.07);
        g.endFill();
    } else if (idx === 11) {
        // Aeon: парящий светлый гексагон с двойным контуром и частицами (Fractal).
        const fl = Math.sin(t * 1.8) * r * 0.05;
        g.beginFill(dark, 0.22);
        g.drawCircle(0, fl * 0.5, r * 1.4);
        g.endFill();
        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(col, 1);
        hexPath(g, r * 0.9, Math.PI / 6);
        g.endFill();
        g.lineStyle(0);
        g.beginFill(0xffffff, 0.16);
        hexPath(g, r * 0.9, Math.PI / 6);
        g.endFill();
        g.lineStyle(2, light, 0.9);
        hexPath(g, r * 0.6, 0);
        g.lineStyle(0);
        g.beginFill(light, 1);
        hexPath(g, r * 0.3, Math.PI / 6);
        g.endFill();
        g.beginFill(0xffffff, 1);
        g.drawCircle(0, fl, r * 0.08);
        g.endFill();
        g.beginFill(light, 0.9);
        for (let i = 0; i < 4; i++) {
            const a = i * Math.PI / 2 + t * 0.6;
            const rad = r * 1.2 + Math.sin(t * 2 + i) * r * 0.08;
            g.drawCircle(Math.cos(a) * rad, Math.sin(a) * rad * 0.6 + fl, r * 0.045);
        }
        g.endFill();
    } else if (idx === 12) {
        // Rune: фиолетовый гексагон-артефакт с руной и орбитальными искрами (Fractal).
        g.beginFill(dark, 0.22);
        g.drawCircle(0, 0, r * 1.4);
        g.endFill();
        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(col, 1);
        hexPath(g, r * 0.95, 0);
        g.endFill();
        g.lineStyle(0);
        g.beginFill(dark, 0.35);
        hexPath(g, r * 0.62, 0);
        g.endFill();
        g.lineStyle(2.5, 0xffffff, 0.95);
        g.moveTo(0, -r * 0.5); g.lineTo(r * 0.28, 0);
        g.lineTo(0, r * 0.5); g.lineTo(-r * 0.28, 0);
        g.closePath();
        g.moveTo(-r * 0.28, 0); g.lineTo(r * 0.28, 0);
        g.lineStyle(0);
        g.beginFill(light, 1);
        for (let i = 0; i < 3; i++) {
            const a = t * 1.4 + i * (Math.PI * 2 / 3);
            g.drawCircle(Math.cos(a) * r * 1.15, Math.sin(a) * r * 1.15, r * 0.07);
        }
        g.endFill();
    } else if (idx === 13) {
        // Cog: шестерёнка с восемью зубьями, осью и болтами. Позже станет
        // реальным мобом и будет дропать Cog-петаль.
        g.beginFill(dark, 0.22);
        g.drawCircle(0, 0, r * 1.4);
        g.endFill();
        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(col, 1);
        const teeth = 8;
        for (let i = 0; i < teeth * 2; i++) {
            const a = i * Math.PI / teeth - Math.PI / 2;
            const rr = (i % 2 === 0) ? r : r * 0.74;
            const x = Math.cos(a) * rr, y = Math.sin(a) * rr;
            if (i === 0) g.moveTo(x, y);
            else g.lineTo(x, y);
        }
        g.closePath();
        g.endFill();
        g.lineStyle(0);
        g.beginFill(dark, 0.3);
        g.drawCircle(0, 0, r * 0.62);
        g.endFill();
        g.lineStyle(2.5, light, 0.9);
        g.beginFill(col, 1);
        g.drawCircle(0, 0, r * 0.32);
        g.endFill();
        g.beginFill(0x0a0f15, 0.9);
        g.drawCircle(0, 0, r * 0.14);
        g.endFill();
        g.lineStyle(0);
        g.beginFill(dark, 1);
        for (let i = 0; i < 4; i++) {
            const a = i * Math.PI / 2 + Math.PI / 4;
            g.drawCircle(Math.cos(a) * r * 0.47, Math.sin(a) * r * 0.47, r * 0.05);
        }
        g.endFill();
    } else if (idx === 14) {
        // Comet: прежний вид — светящееся ядро-голова впереди (+x) и длинный
        // хвост-конус позади. Анимация «живая»: хвост мягко мерцает, из-под
        // головы срываются искры и уплывают по хвосту, ореол дышит.
        // При «агро» головой поворачивается на игрока (tick).
        g.beginFill(dark, 0.15);
        g.drawCircle(r * 0.2, 0, r * 1.4);
        g.endFill();
        g.lineStyle(0);
        const tailFlick = 0.65 + 0.35 * Math.sin(t * 5);
        g.beginFill(col, 0.35 * tailFlick);
        g.moveTo(r * 0.35, -r * 0.42);
        g.lineTo(-r * 1.35, 0);
        g.lineTo(r * 0.35, r * 0.42);
        g.closePath();
        g.endFill();
        g.beginFill(col, 0.65 * tailFlick);
        g.moveTo(r * 0.35, -r * 0.22);
        g.lineTo(-r * 0.9, 0);
        g.lineTo(r * 0.35, r * 0.22);
        g.closePath();
        g.endFill();
        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(light, 1);
        g.drawCircle(r * 0.5, 0, r * 0.45);
        g.endFill();
        g.lineStyle(0);
        g.beginFill(0xffffff, 1);
        g.drawCircle(r * 0.62, -r * 0.08, r * 0.16);
        g.endFill();
        // Искры, срывающиеся с головы и уплывающие по хвосту.
        for (let i = 0; i < 6; i++) {
            const f = (i / 6 + t * 0.05) % 1;
            const sx = r * 0.3 - r * 1.7 * f;
            const sy = Math.sin(t * 4 + i * 2.1) * r * 0.06 * (0.3 + f);
            g.beginFill(0xffffff, 0.8 * (1 - f));
            g.drawCircle(sx, sy, r * 0.045 * (1 - f * 0.5));
            g.endFill();
        }
        // Трепещущий ореол вокруг головы — «заряд» при приближении игрока.
        const cometGp = 0.75 + 0.25 * Math.sin(t * 5);
        g.lineStyle(2.5, light, 0.55 * (0.5 + 0.5 * Math.sin(t * 3)));
        g.drawCircle(r * 0.5, 0, r * (0.62 + 0.1 * cometGp));
        g.lineStyle(0);
    } else if (idx === 15) {
        // Beam: излучатель позади, концентрированный луч вперёд (+x) с
        // пульсацией. При «агро» цель поворота — голова-луч (tick).
        const gp = 0.85 + Math.sin(t * 3) * 0.15;
        g.beginFill(dark, 0.15);
        g.drawCircle(-r * 0.7, 0, r * 1.15);
        g.endFill();
        g.lineStyle(0);
        g.beginFill(col, 0.8 * gp);
        g.moveTo(-r * 0.6, -r * 0.18);
        g.lineTo(r * 1.35, -r * 0.05);
        g.lineTo(r * 1.35, r * 0.05);
        g.lineTo(-r * 0.6, r * 0.18);
        g.closePath();
        g.endFill();
        g.beginFill(0xffffff, 0.95);
        g.moveTo(-r * 0.6, -r * 0.07);
        g.lineTo(r * 1.35, 0);
        g.lineTo(-r * 0.6, r * 0.07);
        g.closePath();
        g.endFill();
        g.lineStyle(3, 0x000000, 0.55);
        g.beginFill(dark, 1);
        g.drawCircle(-r * 0.75, 0, r * 0.4);
        g.endFill();
        g.lineStyle(0);
        g.beginFill(light, 1);
        g.drawCircle(-r * 0.75, 0, r * 0.22);
        g.endFill();
        g.beginFill(0xffffff, 1);
        g.drawCircle(-r * 0.75, 0, r * 0.09);
        g.endFill();
        // «Голова» — электрический наконечник в конце луча.
        const er = r * 0.30; // амплитуда искр
        // 1) Пульсирующее электрическое сияние вокруг кончика.
        g.lineStyle(5, light, 0.35 + 0.3 * Math.sin(t * 9));
        g.beginFill(0xffffff, 0.22 + 0.18 * Math.sin(t * 11));
        g.drawCircle(r * 1.42, 0, er * 2.2 + er * 0.6 * Math.sin(t * 8));
        g.endFill();
        g.lineStyle(0);
        // 2) Дрожащие разряды-молнии, прыгающие с кончика вперёд (веером).
        for (let k = 0; k < 4; k++) {
            const flick = Math.sin(t * 21 + k * 2.9);
            const ang = (-0.85 + k * 0.55) + flick * 0.10;
            const lx = Math.cos(ang), ly = Math.sin(ang);
            const px = -ly, py = lx; // перпендикуляр к направлению разряда
            g.lineStyle(3.5 - k * 0.6, k === 0 ? 0xffffff : light, (0.85 - k * 0.15) * (0.7 + 0.3 * Math.abs(flick)));
            g.moveTo(r * 1.40, 0);
            for (let s = 1; s <= 5; s++) {
                const f = s / 5;
                const wob = Math.sin(t * 27 + k * 5.1 + s * 9.7) * er * (1 - f * 0.45);
                g.lineTo(r * 1.40 + lx * r * 0.6 * f + px * wob, ly * r * 0.6 * f + py * wob);
            }
        }
        // 3) Яркое ядро разряда на самом кончике.
        g.lineStyle(3, 0xffffff, 0.95);
        g.beginFill(light, 1);
        g.drawCircle(r * 1.40, 0, er * 0.55 + er * 0.2 * Math.sin(t * 13));
        g.endFill();
        g.beginFill(0xffffff, 1);
        g.drawCircle(r * 1.40, 0, er * 0.30);
        g.endFill();
        g.lineStyle(0);
    } else if (idx === 16) {
        // Prism: чёткий продолговатый ромб-«стрела» — острый нос (+X) показывает
        // направление движения (поворот делает generic-rotation по dirX/dirY).
        // БЕЗ круглой ауры: только сам кристалл с фасетами и деталями.
        const nose = r * 1.08, side = r * 0.85, tail = r * 0.68;
        // Корпус.
        g.lineStyle(3, 0x000000, 0.6);
        g.beginFill(col, 1);
        g.moveTo(nose, 0);
        g.lineTo(0, -side);
        g.lineTo(-tail, 0);
        g.lineTo(0, side);
        g.closePath();
        g.endFill();
        g.lineStyle(0);
        // Нижняя фасета темнее (верхний полусвет — как у гранёного кристалла).
        g.beginFill(dark, 0.32);
        g.moveTo(nose, 0);
        g.lineTo(0, side);
        g.lineTo(-tail, 0);
        g.closePath();
        g.endFill();
        // Внутренний контур-грань (внутри силуэта).
        const ins = 0.62;
        g.lineStyle(1.6, light, 0.5);
        g.moveTo(nose * ins, 0);
        g.lineTo(0, -side * ins);
        g.lineTo(-tail * ins, 0);
        g.lineTo(0, side * ins);
        g.closePath();
        // Хребет-ось от хвоста к носу.
        g.lineStyle(1.4, light, 0.65);
        g.moveTo(-tail * 0.9, 0);
        g.lineTo(nose * ins, 0);
        // Стрелка-шеврон у носа (подчёркивает направление).
        g.lineStyle(2, 0xffffff, 0.85);
        g.moveTo(nose * 0.44, -r * 0.3);
        g.lineTo(nose * 0.72, 0);
        g.lineTo(nose * 0.44, r * 0.3);
        // Бегущий блик по верхней кромке (нос → бок → хвост).
        const sEdge = (t * 0.7) % 1;
        let ex, ey;
        if (sEdge < 0.5) {
            ex = nose + (0 - nose) * sEdge * 2;
            ey = -side * sEdge * 2;
        } else {
            const f = (sEdge - 0.5) * 2;
            ex = -tail * f;
            ey = -side * (1 - f);
        }
        g.beginFill(0xffffff, 0.9);
        g.drawCircle(ex, ey, r * 0.05);
        g.endFill();
        // Пульсирующее ядро ближе к носу.
        g.beginFill(0xffffff, 0.85);
        g.drawCircle(r * 0.12, 0, r * (0.1 + 0.03 * Math.sin(t * 3.4)));
        g.endFill();
        g.beginFill(0xffffff, 1);
        g.drawCircle(r * 0.12, 0, r * 0.04);
        g.endFill();
    } else if (idx === 20) {
            // Arrowhead: наконечник со шевроном, задающий направление.
            g.lineStyle(3, 0x000000, 0.55);
            g.beginFill(col, 1);
            g.moveTo(r * 1.2, 0);
            g.lineTo(-r * 0.8, -r * 0.8);
            g.lineTo(-r * 0.35, 0);
            g.lineTo(-r * 0.8, r * 0.8);
            g.closePath();
            g.endFill();
            const blink = 0.4 + 0.6 * Math.abs(Math.sin(t * 3));
            g.lineStyle(2.5, 0xffffff, blink);
            g.moveTo(r * 0.35, -r * 0.4); g.lineTo(r * 0.75, 0); g.lineTo(r * 0.35, r * 0.4);
            g.lineStyle(0);
            g.beginFill(light, 0.9);
            g.drawCircle(-r * 0.4, 0, r * 0.12);
            g.endFill();
        } else if (idx === 21) {
            // Shardwing: два крыла-осколка машут (масштаб по вертикали).
            const flap = Math.sin(t * 4.2);
            g.lineStyle(2.5, 0x000000, 0.55);
            g.beginFill(col, 1);
            g.moveTo(-r * 0.2, 0);
            g.lineTo(r * 1.1, -r * (0.35 + 0.45 * flap));
            g.lineTo(r * 0.25, 0);
            g.closePath();
            g.endFill();
            g.beginFill(dark, 1);
            g.moveTo(-r * 0.2, 0);
            g.lineTo(r * 1.1, r * (0.35 + 0.45 * flap));
            g.lineTo(r * 0.25, 0);
            g.closePath();
            g.endFill();
            g.lineStyle(0);
            g.beginFill(light, 0.95);
            g.drawCircle(-r * 0.45, 0, r * 0.3);
            g.endFill();
            g.beginFill(0xffffff, 0.9);
            g.drawCircle(-r * 0.5, -r * 0.08, r * 0.1);
            g.endFill();
        } else if (idx === 26) {
            // Kite: воздушный змей с волнистым хвостом-лентой.
            g.lineStyle(4, light, 0.8);
            let hx = 0, hy = r * 0.9;
            g.moveTo(0, r * 0.7);
            for (let i = 1; i <= 4; i++) {
                hx = Math.sin(t * 3 + i) * r * 0.22;
                hy = r * 0.7 + i * r * 0.3;
                g.lineTo(hx, hy);
            }
            g.lineStyle(3, 0x000000, 0.55);
            g.beginFill(col, 1);
            g.moveTo(0, -r * 1.1);
            g.lineTo(r * 0.7, 0);
            g.lineTo(0, r * 0.75);
            g.lineTo(-r * 0.7, 0);
            g.closePath();
            g.endFill();
            g.lineStyle(1.5, light, 0.8);
            g.moveTo(0, -r * 1.1); g.lineTo(0, r * 0.75);
            g.moveTo(-r * 0.7, 0); g.lineTo(r * 0.7, 0);
            g.lineStyle(0);
            g.beginFill(0xffffff, 0.8);
            g.drawCircle(0, -r * 0.2, r * 0.1);
            g.endFill();
        } else if (idx === 39) {
            // Frame: рамка с парящим внутренним квадратом.
            g.lineStyle(6, col, 1);
            g.drawRect(-r * 0.9, -r * 0.9, r * 1.8, r * 1.8);
            const wob = Math.sin(t * 1.8) * r * 0.12;
            g.lineStyle(2.5, light, 0.9);
            g.drawRect(-r * 0.4 + wob, -r * 0.4 - wob, r * 0.8, r * 0.8);
            g.lineStyle(0);
            g.beginFill(0xffffff, 0.5 + 0.3 * Math.sin(t * 3));
            g.drawCircle(-r * 0.4 + wob + r * 0.4, -r * 0.4 - wob + r * 0.4, r * 0.1);
            g.endFill();
        // === TestRoom 21-50: 30 НОВЫХ концептов с анимациями ===
        // Point-зона (51-65, зелёная палитра, «мир точек») и Pentagon-зона
        // (66-80, синяя палитра, «мир пятиугольников»). Все дизайны «живые»:
        // анимация берётся из performance.now() (t).
        } else if (idx === 51) {
            // Orbit: ядро-точка с двумя наклонными эллиптическими орбитами и
            // бегущими по ним спутниками-точками (Point).
            g.lineStyle(0);
            g.beginFill(col, 0.14);
            g.drawCircle(0, 0, r * 1.3);
            g.endFill();
            const o1 = t * 0.9, o2 = -t * 0.6;
            g.lineStyle(2, light, 0.5);
            g.drawEllipse(0, 0, r * 1.0, r * 0.42);
            g.lineStyle(2, light, 0.35);
            g.drawEllipse(0, 0, r * 0.42, r * 1.0);
            g.lineStyle(0);
            g.beginFill(0xffffff, 1);
            g.drawCircle(Math.cos(o1) * r * 1.0, Math.sin(o1) * r * 0.42, r * 0.09);
            g.drawCircle(Math.cos(o2) * r * 0.42, Math.sin(o2) * r * 1.0, r * 0.07);
            g.endFill();
            g.lineStyle(2.5, 0x000000, 0.55);
            g.beginFill(col, 1);
            g.drawCircle(0, 0, r * 0.34);
            g.endFill();
            g.lineStyle(0);
            g.beginFill(0xffffff, 0.95);
            g.drawCircle(-r * 0.09, -r * 0.09, r * 0.09);
            g.endFill();
        } else if (idx === 52) {
            // Pulse: пульсирующее ядро с волной, расходящейся по кругу (Point).
            const ph = (t * 0.8) % 1;
            g.lineStyle(0);
            g.beginFill(col, 0.12);
            g.drawCircle(0, 0, r * 1.3);
            g.endFill();
            g.lineStyle(3, light, (1 - ph) * 0.7);
            g.drawCircle(0, 0, r * (0.4 + ph * 0.9));
            g.lineStyle(0);
            const gg = 0.55 + 0.45 * Math.sin(t * 4);
            g.lineStyle(2.5, 0x000000, 0.55);
            g.beginFill(col, 1);
            g.drawCircle(0, 0, r * 0.45 * (0.85 + 0.15 * gg));
            g.endFill();
            g.lineStyle(0);
            g.beginFill(0xffffff, 0.5 + 0.4 * gg);
            g.drawCircle(0, 0, r * 0.16 * (0.7 + 0.3 * gg));
            g.endFill();
        } else if (idx === 53) {
            // Splitter: шарик циклично делится на две половинки и снова
            // соединяется (Point).
            const cyc = 0.5 + 0.5 * Math.sin(t * 1.8);
            const sep = r * (0.05 + cyc * 0.42);
            g.lineStyle(0);
            g.beginFill(dark, 0.2);
            g.drawCircle(0, r * 0.15, r * 0.95);
            g.endFill();
            g.lineStyle(2.5, 0x000000, 0.55);
            g.beginFill(col, 1);
            g.drawCircle(-sep, 0, r * 0.5 * (1 - cyc * 0.12));
            g.endFill();
            g.beginFill(light, 1);
            g.drawCircle(sep, 0, r * 0.5 * (1 - cyc * 0.12));
            g.endFill();
            g.lineStyle(0);
            g.beginFill(0x0e2a1c, 0.9);
            g.drawCircle(-sep - r * 0.14, -r * 0.1, r * 0.08);
            g.drawCircle(sep + r * 0.14, -r * 0.1, r * 0.08);
            g.endFill();
            if (cyc < 0.75) {
                g.beginFill(col, (0.75 - cyc) * 1.2);
                g.drawEllipse(0, 0, r * (0.95 - cyc * 0.7), r * 0.34);
                g.endFill();
            }
        } else if (idx === 54) {
            // Coil: пружинка сжимается-разжимается, на концах капельки (Point).
            const amp = r * (0.5 + 0.25 * Math.sin(t * 2.2));
            g.lineStyle(4, col, 1);
            for (let i = 0; i <= 24; i++) {
                const fy = -r + (i / 24) * r * 2;
                const fx = Math.sin((i / 24) * Math.PI * 5 + t * 1.8) * amp;
                if (i === 0) g.moveTo(fx, fy); else g.lineTo(fx, fy);
            }
            g.lineStyle(0);
            g.beginFill(light, 1);
            g.drawCircle(Math.sin(t * 1.8) * amp, -r, r * 0.22);
            g.endFill();
            g.beginFill(0xffffff, 0.8);
            g.drawCircle(0, r, r * 0.14);
            g.endFill();
        } else if (idx === 55) {
            // Comet: мини-комета — светящаяся голова и мерцающий хвост из
            // гаснущих точек (Point).
            const fl = 0.55 + 0.45 * Math.sin(t * 5);
            g.lineStyle(0);
            g.beginFill(col, 0.15 * fl);
            g.drawCircle(r * 0.4, 0, r * 1.2);
            g.endFill();
            for (let i = 1; i <= 7; i++) {
                const f = i / 7;
                const px = r * 0.35 - f * r * 1.35 + Math.sin(t * 3 + i * 1.7) * r * 0.05;
                const py = Math.sin(t * 4 + i * 2.3) * r * 0.08;
                g.beginFill(0xffffff, (1 - f) * 0.85 * (0.6 + 0.4 * fl));
                g.drawCircle(px, py, r * 0.06 * (1 - f * 0.5));
                g.endFill();
            }
            g.lineStyle(2.5, 0x000000, 0.55);
            g.beginFill(light, 1);
            g.drawCircle(r * 0.45, 0, r * 0.4);
            g.endFill();
            g.lineStyle(0);
            g.beginFill(0xffffff, 1);
            g.drawCircle(r * 0.55, -r * 0.08, r * 0.13);
            g.endFill();
        } else if (idx === 56) {
            // Puffer: газовый шар, который «дышит»: полосы плывут, ореол
            // пульсирует (Point).
            const gg = 0.5 + 0.5 * Math.sin(t * 1.6);
            const rr = r * (0.85 + gg * 0.16);
            g.lineStyle(0);
            g.beginFill(col, 0.1);
            g.drawCircle(0, 0, rr * 1.4);
            g.endFill();
            g.lineStyle(2.5, 0x000000, 0.55);
            g.beginFill(col, 1);
            g.drawCircle(0, 0, rr);
            g.endFill();
            g.lineStyle(0);
            g.beginFill(dark, 0.45);
            g.drawEllipse(0, rr * 0.18, rr, rr * 0.65);
            g.endFill();
            g.beginFill(light, 0.35);
            g.drawEllipse(0, -rr * 0.35, rr * 0.7, rr * 0.26 + Math.sin(t * 2) * rr * 0.05);
            g.endFill();
            g.beginFill(0xffffff, 0.5);
            g.drawCircle(-rr * 0.3, -rr * 0.32, rr * 0.16);
            g.endFill();
            const bx = ((t * 0.5) % 2 - 1) * rr;
            g.lineStyle(2, 0xffffff, 0.5);
            g.moveTo(bx, -rr * 0.05); g.lineTo(bx + rr * 0.2, rr * 0.05);
            g.lineStyle(0);
        } else if (idx === 57) {
            // Triplet: три точки в живом треугольнике — вращаются и «дышат» (Point).
            const rot = t * 0.7;
            const pu = r * (0.7 + 0.12 * Math.sin(t * 2.6));
            g.lineStyle(2, light, 0.4);
            for (let i = 0; i < 3; i++) {
                const a0 = rot + i * Math.PI * 2 / 3;
                g.beginFill(col, 0.35);
                g.drawCircle(Math.cos(a0) * pu, Math.sin(a0) * pu, r * 0.18);
                g.endFill();
            }
            g.lineStyle(2.5, 0x000000, 0.5);
            for (let i = 0; i < 3; i++) {
                const a0 = rot + i * Math.PI * 2 / 3;
                g.beginFill(i === 0 ? col : light, 1);
                g.drawCircle(Math.cos(a0) * pu, Math.sin(a0) * pu, r * 0.3);
                g.endFill();
            }
            g.beginFill(0xffffff, 0.85);
            g.drawCircle(0, 0, r * 0.1);
            g.endFill();
        } else if (idx === 58) {
            // Comb: сотовая точка — шестиугольник с ячейками, загорающимися
            // волной (Point).
            g.lineStyle(0);
            g.beginFill(col, 0.12);
            g.drawCircle(0, 0, r * 1.25);
            g.endFill();
            g.lineStyle(2.5, col, 1);
            hexPath(g, r * 0.98, 0);
            g.lineStyle(0);
            const cellR = r * 0.24;
            for (let i = 0; i < 6; i++) {
                const a = Math.PI / 6 + i * Math.PI / 3;
                const cx = Math.cos(a) * r * 0.55, cy = Math.sin(a) * r * 0.55;
                const ph = (t * 0.5 + i / 6) % 1;
                g.beginFill(light, 0.2 + 0.8 * ph * ph);
                hexPath(g, cellR, a);
                g.endFill();
            }
            g.lineStyle(2, 0x000000, 0.55);
            g.beginFill(col, 1);
            hexPath(g, cellR, 0);
            g.endFill();
            g.lineStyle(0);
            g.beginFill(0xffffff, 0.8);
            hexPath(g, cellR * 0.35, 0);
            g.endFill();
        } else if (idx === 59) {
            // Blinker: глазок-точка моргает — веки сжимаются, зрачок светится (Point).
            const blink = Math.max(0, Math.sin(t * 0.9));
            const open = 0.35 + 0.65 * blink;
            g.lineStyle(0);
            g.beginFill(col, 0.12);
            g.drawCircle(0, 0, r * 1.25);
            g.endFill();
            g.lineStyle(2.5, 0x000000, 0.55);
            g.beginFill(col, 1);
            g.drawEllipse(0, 0, r * 0.9, r * 0.9 * open);
            g.endFill();
            g.beginFill(0xffffff, 1);
            g.drawEllipse(0, 0, r * 0.55, r * 0.55 * open);
            g.endFill();
            g.lineStyle(0);
            g.beginFill(0x0e2a1c, 1);
            g.drawCircle(0, 0, r * 0.24 * open + r * 0.03);
            g.endFill();
            if (blink > 0.3) {
                g.beginFill(0xffffff, 1);
                g.drawCircle(-r * 0.2, -r * 0.15, r * 0.1);
                g.endFill();
            }
            g.beginFill(dark, 0.9);
            g.drawEllipse(0, r * 0.55 * (1 - blink), r * 1.0, r * 0.5);
            g.endFill();
        } else if (idx === 60) {
            // Jelly: дрожащая капля-желе — контур пляшет от времени (Point).
            g.lineStyle(2.5, 0x000000, 0.5);
            g.beginFill(col, 0.9);
            for (let i = 0; i <= 22; i++) {
                const a = (i / 22) * Math.PI * 2;
                const rr2 = r * (0.85 + 0.09 * Math.sin(a * 3 + t * 3.2) * Math.sin(a * 2 + t * 2.1));
                const px = Math.cos(a) * rr2, py = Math.sin(a) * rr2 * (1 + 0.08 * Math.sin(t * 1.7));
                if (i === 0) g.moveTo(px, py); else g.lineTo(px, py);
            }
            g.closePath();
            g.endFill();
            g.lineStyle(0);
            g.beginFill(0xffffff, 0.4);
            g.drawCircle(-r * 0.3, -r * 0.3, r * 0.2);
            g.endFill();
            g.beginFill(0xffffff, 0.9);
            g.drawCircle(-r * 0.28, -r * 0.32, r * 0.07);
            g.endFill();
        } else if (idx === 61) {
            // Tick: клещ — шарик с шевелящимися лапками-щетинками (Point).
            for (let i = 0; i < 8; i++) {
                const a = (i / 8) * Math.PI * 2;
                const wig = Math.sin(t * 3.5 + i * 1.8) * 0.35;
                g.lineStyle(2.5, dark, 0.95);
                g.moveTo(Math.cos(a) * r * 0.45, Math.sin(a) * r * 0.45);
                g.lineTo(Math.cos(a + wig) * r * 0.95, Math.sin(a + wig) * r * 0.95);
            }
            g.lineStyle(2.5, 0x000000, 0.55);
            g.beginFill(col, 1);
            g.drawCircle(0, 0, r * 0.55);
            g.endFill();
            g.lineStyle(0);
            g.beginFill(dark, 0.45);
            g.drawCircle(0, r * 0.12, r * 0.5);
            g.endFill();
            g.beginFill(0xffffff, 0.9);
            g.drawCircle(-r * 0.14, -r * 0.14, r * 0.09);
            g.endFill();
        } else if (idx === 62) {
            // Dewdrop: росинка с перетекающим бликом (Point).
            g.lineStyle(2.5, 0x000000, 0.55);
            g.beginFill(col, 1);
            g.moveTo(0, -r);
            g.quadraticCurveTo(r * 0.85, -r * 0.1, r * 0.6, r * 0.8);
            g.quadraticCurveTo(0, r * 1.1, -r * 0.6, r * 0.8);
            g.quadraticCurveTo(-r * 0.85, -r * 0.1, 0, -r);
            g.endFill();
            g.lineStyle(0);
            const spec = (t * 0.5) % 1;
            g.beginFill(0xffffff, 0.8 * (1 - spec));
            g.drawCircle(-r * 0.25 + spec * r * 0.5, -r * 0.25 + spec * r * 0.6, r * 0.1);
            g.endFill();
            g.beginFill(0xffffff, 0.35);
            g.drawCircle(-r * 0.15, -r * 0.3, r * 0.16);
            g.endFill();
        } else if (idx === 63) {
            // Clover: трилистник из точек — листики «дышат» и вращаются (Point).
            const rot = t * 0.5;
            const breathe = 1 + 0.18 * Math.sin(t * 2.0);
            g.lineStyle(0);
            g.beginFill(col, 0.1);
            g.drawCircle(0, 0, r * 1.3);
            g.endFill();
            for (let i = 0; i < 3; i++) {
                const a = rot + i * Math.PI * 2 / 3;
                g.lineStyle(2.5, 0x000000, 0.5);
                g.beginFill(i === 0 ? col : light, 1);
                g.drawEllipse(Math.cos(a) * r * 0.42, Math.sin(a) * r * 0.42, r * 0.42 * breathe, r * 0.42 * breathe);
                g.endFill();
                g.lineStyle(0);
                g.beginFill(0xffffff, 0.35);
                g.drawCircle(Math.cos(a) * r * 0.42 - r * 0.1, Math.sin(a) * r * 0.42 - r * 0.1, r * 0.12);
                g.endFill();
            }
        } else if (idx === 64) {
            // Vortex: мини-вихрь — кольца точек сбегаются к центру (Point).
            g.lineStyle(0);
            g.beginFill(col, 0.1);
            g.drawCircle(0, 0, r * 1.35);
            g.endFill();
            for (let ring = 0; ring < 3; ring++) {
                const n = 6 + ring * 3;
                const base = t * (0.8 + ring * 0.5) + ring * 1.3;
                for (let i = 0; i < n; i++) {
                    const a = base + (i / n) * Math.PI * 2;
                    const f = (i / n + t * 0.4 * (ring + 1)) % 1;
                    const rad = r * (0.25 + (1 - f) * 0.9) * (1 - ring * 0.18);
                    g.beginFill(ring === 1 ? 0xffffff : light, 0.4 + 0.6 * f);
                    g.drawCircle(Math.cos(a) * rad, Math.sin(a) * rad, r * 0.07);
                    g.endFill();
                }
            }
            g.beginFill(0xffffff, 0.9);
            g.drawCircle(0, 0, r * 0.13);
            g.endFill();
        } else if (idx === 65) {
            // Spore: спора с пульсирующими шипами и вылетающими микро-точками.
            const gg = 0.5 + 0.5 * Math.sin(t * 2.4);
            g.lineStyle(0);
            g.beginFill(col, 0.1 + 0.1 * gg);
            g.drawCircle(0, 0, r * 1.35);
            g.endFill();
            for (let i = 0; i < 10; i++) {
                const a = (i / 10) * Math.PI * 2;
                const len = r * (0.8 + 0.28 * gg + 0.12 * Math.sin(t * 3.4 + i * 1.9));
                g.lineStyle(2.5, gg > 0.5 ? light : dark, 0.8);
                g.moveTo(Math.cos(a) * r * 0.45, Math.sin(a) * r * 0.45);
                g.lineTo(Math.cos(a) * len, Math.sin(a) * len);
            }
            g.lineStyle(2.5, 0x000000, 0.55);
            g.beginFill(col, 1);
            g.drawCircle(0, 0, r * 0.5);
            g.endFill();
            g.lineStyle(0);
            g.beginFill(dark, 0.4);
            g.drawCircle(0, r * 0.1, r * 0.44);
            g.endFill();
            for (let i = 0; i < 3; i++) {
                const f = ((t * 0.35 + i / 3) % 1);
                g.beginFill(0xffffff, (1 - f) * 0.8);
                g.drawCircle(Math.cos(i * 2.4) * r * (0.8 + f * 1.2), Math.sin(i * 2.4) * r * (0.5 + f * 1.2), r * 0.05 * (1 - f));
                g.endFill();
            }
        } else if (idx === 66) {
            // Pyramid: пятигранная пирамида — «вращается» панорамированием
            // вершин, в центре пульсирующее ядро (Pentagon).
            const rot = t * 0.5;
            const pts = [];
            for (let i = 0; i < 5; i++) {
                const a = rot * 0.6 + i * Math.PI * 2 / 5 - Math.PI / 2;
                const rr = r * 0.95 * (1 + 0.18 * Math.cos(a + t * 0.9));
                pts.push([Math.cos(a) * rr, Math.sin(a) * rr]);
            }
            g.lineStyle(0);
            g.beginFill(col, 0.1);
            g.drawCircle(0, 0, r * 1.3);
            g.endFill();
            g.lineStyle(2.5, 0x000000, 0.55);
            g.beginFill(col, 1);
            g.moveTo(pts[0][0], pts[0][1]);
            for (let i = 1; i <= 5; i++) g.lineTo(pts[i % 5][0], pts[i % 5][1]);
            g.closePath();
            g.endFill();
            g.lineStyle(2.5, light, 0.95);
            g.moveTo(0, 0);
            for (let i = 0; i < 5; i++) g.lineTo(pts[i][0], pts[i][1]);
            g.lineStyle(0);
            g.beginFill(0xffffff, 0.85);
            g.drawCircle(0, 0, r * 0.1);
            g.endFill();
        } else if (idx === 67) {
            // Geode: пятиугольная жеода — тёмная полость с мерцающими
            // кристаллами-зубцами (Pentagon).
            const pent = (rr, rot2) => {
                g.moveTo(Math.cos(rot2) * rr, Math.sin(rot2) * rr);
                for (let i = 1; i <= 5; i++) {
                    const a = rot2 + i * Math.PI * 2 / 5;
                    g.lineTo(Math.cos(a) * rr, Math.sin(a) * rr);
                }
                g.closePath();
            };
            g.lineStyle(0);
            g.beginFill(col, 0.1);
            g.drawCircle(0, 0, r * 1.3);
            g.endFill();
            g.lineStyle(5, dark, 1);
            g.beginFill(0x101a2c, 1);
            pent(r * 0.95, -Math.PI / 2);
            g.endFill();
            g.lineStyle(2, light, 0.6);
            pent(r * 0.72, -Math.PI / 2);
            g.lineStyle(0);
            for (let i = 0; i < 8; i++) {
                const a = (i / 8) * Math.PI * 2 + Math.sin(t * 0.3) * 0.1;
                const gl = 0.3 + 0.7 * Math.abs(Math.sin(t * 1.4 + i * 1.4));
                const bx = Math.cos(a) * r * 0.52, by = Math.sin(a) * r * 0.52;
                g.beginFill(light, gl);
                g.moveTo(bx - r * 0.08, by - r * 0.05);
                g.lineTo(bx + r * 0.08, by - r * 0.05);
                g.lineTo(bx + r * 0.03, by - r * (0.2 + 0.5 * gl));
                g.closePath();
                g.endFill();
            }
            g.beginFill(0xffffff, 0.85);
            g.drawCircle(0, 0, r * 0.1);
            g.endFill();
        } else if (idx === 68) {
            // Star5: пятиконечная звезда — кончики пульсируют, вращается (Pentagon).
            const rot = t * 0.4;
            const pulse = 1 + 0.1 * Math.sin(t * 2.2);
            g.lineStyle(0);
            g.beginFill(col, 0.12);
            g.drawCircle(0, 0, r * 1.35);
            g.endFill();
            g.lineStyle(3, 0x000000, 0.55);
            g.beginFill(col, 1);
            starPath(g, 5, r * pulse, r * 0.42 * pulse, rot - Math.PI / 2);
            g.endFill();
            g.lineStyle(0);
            g.beginFill(dark, 0.3);
            starPath(g, 5, r * 0.66 * pulse, r * 0.3 * pulse, rot - Math.PI / 2);
            g.endFill();
            g.lineStyle(2, light, 0.9);
            g.beginFill(light, 1);
            g.drawCircle(0, 0, r * 0.3);
            g.endFill();
            g.lineStyle(0);
            g.beginFill(0xffffff, 1);
            g.drawCircle(0, 0, r * 0.1);
            g.endFill();
        } else if (idx === 69) {
            // Rotor: пятиугольный ротор — диск и гребные лопасти вращаются (Pentagon).
            const rot = t * 1.2;
            g.lineStyle(0);
            g.beginFill(col, 0.1);
            g.drawCircle(0, 0, r * 1.3);
            g.endFill();
            for (let i = 0; i < 5; i++) {
                const a = rot + i * Math.PI * 2 / 5;
                g.lineStyle(0);
                g.beginFill(light, 0.85);
                g.moveTo(0, 0);
                g.lineTo(Math.cos(a) * r * 0.9, Math.sin(a) * r * 0.9);
                g.lineTo(Math.cos(a + 0.55) * r * 0.42, Math.sin(a + 0.55) * r * 0.42);
                g.closePath();
                g.endFill();
            }
            g.lineStyle(0);
            g.beginFill(dark, 1);
            g.drawCircle(0, 0, r * 0.42);
            g.endFill();
            g.lineStyle(2.5, light, 0.9);
            g.beginFill(col, 1);
            g.drawCircle(0, 0, r * 0.24);
            g.endFill();
            g.lineStyle(0);
            g.beginFill(0xffffff, 0.9);
            g.drawCircle(0, 0, r * 0.08);
            g.endFill();
            for (let i = 0; i < 5; i++) {
                const a = -rot + i * Math.PI * 2 / 5;
                g.beginFill(0xffffff, 0.6);
                g.drawCircle(Math.cos(a) * r * 0.8, Math.sin(a) * r * 0.8, r * 0.04);
                g.endFill();
            }
        } else if (idx === 70) {
            // Mirror: пятиугольное зеркало — по граням переливаются фасеты,
            // по ободу бежит световой блик (Pentagon).
            const rot = t * 0.35;
            const pent = (rr, rot2) => {
                g.moveTo(Math.cos(rot2) * rr, Math.sin(rot2) * rr);
                for (let i = 1; i <= 5; i++) {
                    const a = rot2 + i * Math.PI * 2 / 5;
                    g.lineTo(Math.cos(a) * rr, Math.sin(a) * rr);
                }
                g.closePath();
            };
            g.lineStyle(0);
            g.beginFill(col, 0.12);
            g.drawCircle(0, 0, r * 1.3);
            g.endFill();
            g.lineStyle(3, 0x000000, 0.55);
            g.beginFill(0x1a2740, 1);
            pent(r * 0.95, -Math.PI / 2);
            g.endFill();
            for (let i = 0; i < 5; i++) {
                const a = rot - Math.PI / 2 + i * Math.PI * 2 / 5;
                const gl = 0.2 + 0.5 * Math.abs(Math.sin(t * 1.2 + i * 1.8));
                g.beginFill(light, gl);
                g.moveTo(0, 0);
                g.lineTo(Math.cos(a) * r * 0.95, Math.sin(a) * r * 0.95);
                g.lineTo(Math.cos(a + Math.PI * 2 / 5) * r * 0.95, Math.sin(a + Math.PI * 2 / 5) * r * 0.95);
                g.closePath();
                g.endFill();
            }
            g.lineStyle(3.5, 0xffffff, 0.9 * (1 - ((t * 0.6) % 1)));
            pent(r * 0.82, rot + t * 0.6 - Math.PI / 2);
            g.lineStyle(0);
            g.beginFill(0xffffff, 0.85);
            g.drawCircle(0, 0, r * 0.12);
            g.endFill();
        } else if (idx === 71) {
            // Mandala: пятиугольная мандала — узор вращается в противофазе,
            // кольца пульсируют (Pentagon).
            const inner = t * 0.8, outer = -t * 0.45;
            g.lineStyle(0);
            g.beginFill(col, 0.1);
            g.drawCircle(0, 0, r * 1.3);
            g.endFill();
            g.lineStyle(2, light, 0.5);
            g.drawCircle(0, 0, r * 0.7);
            g.drawCircle(0, 0, r * 0.44);
            g.lineStyle(0);
            for (let i = 0; i < 10; i++) {
                const a = outer + i * Math.PI / 5;
                g.lineStyle(3, i % 2 === 0 ? light : 0xffffff, 0.7);
                g.moveTo(Math.cos(a) * r * 0.7, Math.sin(a) * r * 0.7);
                g.lineTo(Math.cos(a) * r * 0.95, Math.sin(a) * r * 0.95);
            }
            for (let i = 0; i < 5; i++) {
                const a = inner + i * Math.PI * 2 / 5;
                g.lineStyle(0);
                g.beginFill(light, 0.9);
                g.drawCircle(Math.cos(a) * r * 0.58, Math.sin(a) * r * 0.58, r * 0.1 + r * 0.05 * Math.sin(t * 2 + i));
                g.endFill();
            }
            g.lineStyle(2, 0x000000, 0.55);
            g.beginFill(col, 1);
            g.drawCircle(0, 0, r * 0.2);
            g.endFill();
            g.lineStyle(0);
            g.beginFill(0xffffff, 0.85);
            g.drawCircle(0, 0, r * 0.07);
            g.endFill();
        } else if (idx === 72) {
            // Wing: «пента-мотылёк» — пятиугольные крылья машут (Pentagon).
            const flap = Math.sin(t * 3.6);
            const a = 0.6 + 0.5 * flap;
            g.lineStyle(0);
            g.beginFill(col, 0.1);
            g.drawCircle(0, 0, r * 1.3);
            g.endFill();
            for (const s of [1, -1]) {
                g.lineStyle(2.5, 0x000000, 0.5);
                g.beginFill(s === 1 ? col : light, 0.9);
                g.moveTo(0, 0);
                g.lineTo(-r * 0.75, -r * 0.42 * a * s);
                g.lineTo(-r * 0.28, -r * (0.2 + 0.5 * a) * s);
                g.closePath();
                g.endFill();
                g.lineStyle(0);
            }
            g.lineStyle(2.5, 0x000000, 0.55);
            g.beginFill(dark, 1);
            g.drawEllipse(r * 0.08, 0, r * 0.5, r * 0.22);
            g.endFill();
            g.lineStyle(0);
            g.beginFill(light, 1);
            g.drawCircle(r * 0.5, 0, r * 0.12);
            g.endFill();
            g.beginFill(0xffffff, 1);
            g.drawCircle(r * 0.53, -r * 0.04, r * 0.05);
            g.endFill();
        } else if (idx === 73) {
            // Compass: пятиугольный компас — вращающаяся стрелка-игла на
            // светящемся циферблате (Pentagon).
            const rot = t * 0.9;
            const pent = (rr, rot2) => {
                g.moveTo(Math.cos(rot2) * rr, Math.sin(rot2) * rr);
                for (let i = 1; i <= 5; i++) {
                    const a = rot2 + i * Math.PI * 2 / 5;
                    g.lineTo(Math.cos(a) * rr, Math.sin(a) * rr);
                }
                g.closePath();
            };
            g.lineStyle(0);
            g.beginFill(col, 0.12);
            g.drawCircle(0, 0, r * 1.3);
            g.endFill();
            g.lineStyle(3, 0x000000, 0.55);
            g.beginFill(col, 1);
            pent(r * 0.95, -Math.PI / 2);
            g.endFill();
            g.lineStyle(0);
            g.beginFill(0x1a2740, 1);
            pent(r * 0.7, -Math.PI / 2);
            g.endFill();
            g.lineStyle(1.5, light, 0.7);
            for (let i = 0; i < 20; i++) {
                const a = -Math.PI / 2 + i * Math.PI / 10;
                g.moveTo(Math.cos(a) * r * 0.62, Math.sin(a) * r * 0.62);
                g.lineTo(Math.cos(a) * r * 0.68, Math.sin(a) * r * 0.68);
            }
            g.lineStyle(0);
            const np = 0.5 + 0.5 * Math.sin(t * 2);
            g.beginFill(0xffffff, 0.4 + 0.4 * np);
            g.moveTo(Math.cos(rot) * r * 0.5, Math.sin(rot) * r * 0.5);
            g.lineTo(Math.cos(rot + Math.PI) * r * 0.3, Math.sin(rot + Math.PI) * r * 0.3);
            g.lineTo(0, 0);
            g.closePath();
            g.endFill();
            g.lineStyle(5, 0xffffff, 0.95);
            g.moveTo(0, 0); g.lineTo(Math.cos(rot) * r * 0.55, Math.sin(rot) * r * 0.55);
            g.lineStyle(0);
            g.beginFill(0xffffff, 0.9);
            g.drawCircle(0, 0, r * 0.09);
            g.endFill();
        } else if (idx === 74) {
            // Hive5: пятиугольная сота — ячейки загораются волной (Pentagon).
            const pent = (rr, rot2) => {
                g.moveTo(Math.cos(rot2) * rr, Math.sin(rot2) * rr);
                for (let i = 1; i <= 5; i++) {
                    const a = rot2 + i * Math.PI * 2 / 5;
                    g.lineTo(Math.cos(a) * rr, Math.sin(a) * rr);
                }
                g.closePath();
            };
            g.lineStyle(0);
            g.beginFill(col, 0.12);
            g.drawCircle(0, 0, r * 1.3);
            g.endFill();
            g.lineStyle(3, 0x000000, 0.55);
            g.beginFill(col, 1);
            pent(r * 0.98, -Math.PI / 2);
            g.endFill();
            g.lineStyle(0);
            for (let i = 0; i < 5; i++) {
                const a = -Math.PI / 2 + i * Math.PI * 2 / 5;
                const cx = Math.cos(a) * r * 0.5, cy = Math.sin(a) * r * 0.5;
                const ph = (i / 5 + t * 0.4) % 1;
                g.beginFill(0xffd27a, Math.min(1, ph * 1.6));
                g.drawCircle(cx, cy, r * 0.22);
                g.endFill();
                g.beginFill(0xffe08a, Math.max(0, ph * 1.6 - 0.5));
                g.drawCircle(cx, cy, r * 0.12);
                g.endFill();
            }
            g.beginFill(0xffe08a, 0.6 + 0.4 * Math.sin(t * 2));
            g.drawCircle(0, 0, r * 0.16);
            g.endFill();
        } else if (idx === 75) {
            // Gear5: пятиугольная шестерня — зубцы, ось и «скоростные» искры
            // вращаются и мерцают (Pentagon).
            const rot = t * 0.7;
            g.lineStyle(0);
            g.beginFill(col, 0.12);
            g.drawCircle(0, 0, r * 1.3);
            g.endFill();
            g.lineStyle(3, 0x000000, 0.55);
            g.beginFill(col, 1);
            g.moveTo(Math.cos(rot) * r * 0.95, Math.sin(rot) * r * 0.95);
            for (let i = 1; i <= 10; i++) {
                const a = rot + i * Math.PI / 5;
                const rr2 = (i % 2 === 0) ? r * 0.95 : r * 0.72;
                g.lineTo(Math.cos(a) * rr2, Math.sin(a) * rr2);
            }
            g.closePath();
            g.endFill();
            g.lineStyle(0);
            g.beginFill(dark, 0.4);
            g.drawCircle(0, 0, r * 0.55);
            g.endFill();
            g.lineStyle(2.5, light, 0.9);
            g.beginFill(col, 1);
            g.drawCircle(0, 0, r * 0.28);
            g.endFill();
            g.lineStyle(0);
            g.beginFill(0xffffff, 0.9);
            g.drawCircle(0, 0, r * 0.1);
            g.endFill();
            for (let i = 0; i < 5; i++) {
                const a = rot + i * Math.PI * 2 / 5 + Math.PI / 5;
                g.beginFill(0xffffff, 0.5 + 0.5 * Math.sin(t * 4 + i * 2));
                g.drawCircle(Math.cos(a) * r * 1.02, Math.sin(a) * r * 1.02, r * 0.05);
                g.endFill();
            }
        } else if (idx === 76) {
            // Bloom: пятилепестковый каменный цветок — лепестки раскрываются (Pentagon).
            const open = 0.5 + 0.5 * Math.sin(t * 1.5);
            g.lineStyle(0);
            g.beginFill(col, 0.12);
            g.drawCircle(0, 0, r * 1.3);
            g.endFill();
            for (let i = 0; i < 5; i++) {
                const a0 = i * Math.PI * 2 / 5 - Math.PI / 2;
                const a1 = a0 + Math.PI * 2 / 5;
                const len = r * (0.75 + 0.35 * open);
                g.lineStyle(2.5, 0x000000, 0.5);
                g.beginFill(i % 2 === 0 ? col : light, 0.95);
                g.moveTo(0, 0);
                for (let k = 0; k <= 12; k++) {
                    const aa = a0 + (a1 - a0) * k / 12;
                    g.lineTo(Math.cos(aa) * len, Math.sin(aa) * len);
                }
                g.closePath();
                g.endFill();
                g.lineStyle(0);
                const tip = (a0 + a1) / 2;
                g.beginFill(0xffffff, 0.35 * open);
                g.drawCircle(Math.cos(tip) * len, Math.sin(tip) * len, r * 0.09);
                g.endFill();
            }
            g.lineStyle(0);
            g.beginFill(0xfff3b0, 0.8);
            g.drawCircle(0, 0, r * 0.15);
            g.endFill();
            g.beginFill(0xffffff, 0.85);
            g.drawCircle(0, 0, r * 0.07);
            g.endFill();
        } else if (idx === 77) {
            // Ion: пятиугольная ионная ловушка — частицы бегают по орбите с
            // пульсирующим радиусом вокруг ядра (Pentagon).
            const pent = (rr, rot2) => {
                g.moveTo(Math.cos(rot2) * rr, Math.sin(rot2) * rr);
                for (let i = 1; i <= 5; i++) {
                    const a = rot2 + i * Math.PI * 2 / 5;
                    g.lineTo(Math.cos(a) * rr, Math.sin(a) * rr);
                }
                g.closePath();
            };
            g.lineStyle(0);
            g.beginFill(col, 0.12);
            g.drawCircle(0, 0, r * 1.3);
            g.endFill();
            g.lineStyle(2.5, 0x000000, 0.55);
            g.beginFill(0x1a2740, 1);
            pent(r * 0.85, t * 0.3);
            g.endFill();
            g.lineStyle(2, light, 0.6);
            g.drawCircle(0, 0, r * 0.6);
            g.lineStyle(0);
            for (let i = 0; i < 3; i++) {
                const a = t * 1.4 + i * Math.PI * 2 / 3;
                const rad = r * 0.25 + r * 0.35 * (0.5 + 0.5 * Math.sin(t * 2 + i * 2.1));
                const pulse = 0.6 + 0.4 * Math.sin(t * 5 + i * 2);
                g.beginFill(0x9be0ff, pulse);
                g.drawCircle(Math.cos(a) * rad, Math.sin(a) * rad, r * 0.09);
                g.endFill();
            }
            g.beginFill(0xffffff, 0.95);
            g.drawCircle(0, 0, r * 0.1);
            g.endFill();
        } else if (idx === 78) {
            // Rune5: пятиугольный артефакт — внутри вращается светящаяся
            // руна-квадрат, углы пульсируют (Pentagon).
            const rot = t * 0.6;
            const pent = (rr, rot2) => {
                g.moveTo(Math.cos(rot2) * rr, Math.sin(rot2) * rr);
                for (let i = 1; i <= 5; i++) {
                    const a = rot2 + i * Math.PI * 2 / 5;
                    g.lineTo(Math.cos(a) * rr, Math.sin(a) * rr);
                }
                g.closePath();
            };
            g.lineStyle(0);
            g.beginFill(col, 0.12);
            g.drawCircle(0, 0, r * 1.3);
            g.endFill();
            g.lineStyle(3, 0x000000, 0.55);
            g.beginFill(col, 1);
            pent(r * 0.95, rot * 0.25);
            g.endFill();
            g.lineStyle(0);
            g.beginFill(0x16233c, 1);
            pent(r * 0.7, rot * 0.25);
            g.endFill();
            g.lineStyle(3, light, 0.95);
            g.moveTo(Math.cos(rot) * r * 0.4, Math.sin(rot) * r * 0.4);
            for (let i = 1; i <= 4; i++) {
                const a = rot + i * Math.PI / 2;
                g.lineTo(Math.cos(a) * r * 0.4, Math.sin(a) * r * 0.4);
            }
            g.closePath();
            g.lineStyle(0);
            g.beginFill(0xffffff, 0.7);
            g.drawCircle(Math.cos(rot) * r * 0.4, Math.sin(rot) * r * 0.4, r * 0.06);
            g.endFill();
        } else if (idx === 79) {
            // Shard5: пять осколков-лезвий вращаются вокруг ядра и мерцают (Pentagon).
            const rot = t * 0.8;
            g.lineStyle(0);
            g.beginFill(col, 0.12);
            g.drawCircle(0, 0, r * 1.35);
            g.endFill();
            for (let i = 0; i < 5; i++) {
                const a = rot + i * Math.PI * 2 / 5;
                const shin = 0.55 + 0.45 * Math.sin(t * 3 + i * 2.2);
                g.lineStyle(2.5, 0x000000, 0.5);
                g.beginFill(light, 0.35 + 0.5 * shin);
                g.moveTo(Math.cos(a) * r * 0.3, Math.sin(a) * r * 0.3);
                g.lineTo(Math.cos(a + 0.42) * r * 0.9, Math.sin(a + 0.42) * r * 0.9);
                g.lineTo(Math.cos(a - 0.42) * r * 0.9, Math.sin(a - 0.42) * r * 0.9);
                g.closePath();
                g.endFill();
                g.lineStyle(0);
                g.lineStyle(2, 0xffffff, 0.4 + 0.5 * shin);
                g.moveTo(Math.cos(a) * r * 0.3, Math.sin(a) * r * 0.3);
                g.lineTo(Math.cos(a) * r * 0.92, Math.sin(a) * r * 0.92);
                g.lineStyle(0);
            }
            g.lineStyle(2.5, 0x000000, 0.55);
            g.beginFill(col, 1);
            g.drawCircle(0, 0, r * 0.34);
            g.endFill();
            g.lineStyle(0);
            g.beginFill(0xffffff, 0.9);
            g.drawCircle(0, 0, r * 0.1);
            g.endFill();
        } else if (idx === 80) {
            // Sentinel: пятиугольный страж — по ободу бежит «луч сканера»,
            // антенна качается, сенсор сканирует (Pentagon).
            const pent = (rr, rot2) => {
                g.moveTo(Math.cos(rot2) * rr, Math.sin(rot2) * rr);
                for (let i = 1; i <= 5; i++) {
                    const a = rot2 + i * Math.PI * 2 / 5;
                    g.lineTo(Math.cos(a) * rr, Math.sin(a) * rr);
                }
                g.closePath();
            };
            const sweep = t * 0.9;
            g.lineStyle(0);
            g.beginFill(col, 0.12);
            g.drawCircle(0, 0, r * 1.35);
            g.endFill();
            g.lineStyle(3.5, dark, 1);
            g.moveTo(0, r * 0.4);
            g.lineTo(Math.sin(t * 1.2) * r * 0.3, -r * 0.8);
            g.lineStyle(0);
            g.beginFill(0xffffff, 0.9);
            g.drawCircle(Math.sin(t * 1.2) * r * 0.3, -r * 0.85, r * 0.08);
            g.endFill();
            g.lineStyle(3, 0x000000, 0.55);
            g.beginFill(col, 1);
            pent(r * 0.9, -Math.PI / 2);
            g.endFill();
            g.lineStyle(0);
            g.beginFill(dark, 0.35);
            pent(r * 0.62, -Math.PI / 2);
            g.endFill();
            g.lineStyle(2, light, 0.6);
            g.drawCircle(0, 0, r * 0.75);
            g.lineStyle(0);
            const sx = Math.cos(sweep) * r * 0.75, sy = Math.sin(sweep) * r * 0.75;
            const sz = 0.5 + 0.5 * Math.sin(t * 6);
            g.beginFill(0xffe08a, 0.35 + 0.4 * sz);
            g.drawCircle(sx, sy, r * 0.09);
            g.endFill();
            g.beginFill(0xffffff, 0.9);
            g.drawCircle(0, 0, r * 0.22);
            g.endFill();
            g.beginFill(0x16233c, 0.85 + 0.15 * Math.sin(t * 3));
            g.drawCircle(Math.sin(t * 0.8) * r * 0.06, 0, r * 0.09);
            g.endFill();
            g.beginFill(0xffffff, 0.85);
            g.drawCircle(-r * 0.08, -r * 0.08, r * 0.03);
            g.endFill();
        } else if (idx === 81) {
            // Sprout: росток — стебель растёт из точки, два листика «дышат» (Point).
            const gr = 0.5 + 0.5 * Math.sin(t * 2.2);
            g.beginFill(col, 0.1 + 0.08 * gr);
            g.drawCircle(0, 0, r * 1.3);
            g.endFill();
            g.lineStyle(2.5, 0x000000, 0.5);
            g.beginFill(dark, 1);
            g.moveTo(0, r * 0.7);
            g.lineTo(-r * 0.08, r * 0.1);
            g.lineTo(r * 0.08, r * 0.1);
            g.closePath();
            g.endFill();
            g.lineStyle(2.5, dark, 1);
            g.moveTo(0, r * 0.1);
            g.lineTo(0, -r * 0.42);
            g.lineStyle(0);
            const leafA = 0.55 + 0.45 * Math.sin(t * 2.0);
            g.beginFill(col, 1);
            g.drawEllipse(-r * 0.28, -r * 0.42, r * 0.3 * leafA, r * 0.16);
            g.endFill();
            g.beginFill(light, 1);
            g.drawEllipse(r * 0.28, -r * 0.42, r * 0.3 * (2 - leafA), r * 0.16);
            g.endFill();
            g.beginFill(0xffffff, 0.95);
            g.drawCircle(0, -r * 0.58 + (1 - gr) * r * 0.06, r * 0.09 * (1 + 0.25 * gr));
            g.endFill();
        } else if (idx === 82) {
            // Fern: веер папоротника — сегменты поочерёдно раскрываются (Point).
            g.beginFill(col, 0.1);
            g.drawCircle(0, 0, r * 1.3);
            g.endFill();
            g.lineStyle(2.5, dark, 1);
            g.moveTo(0, r * 0.75);
            g.lineTo(0, -r * 0.05);
            g.lineStyle(0);
            for (let i = 0; i < 5; i++) {
                const s = Math.sin(t * 2.0 - i * 0.9) * 0.35;
                for (const sgn of [1, -1]) {
                    g.lineStyle(2.5, i % 2 ? light : col, 0.9);
                    g.moveTo(0, -r * 0.05);
                    g.lineTo(sgn * (r * 0.18 + i * r * 0.12), -r * 0.05 - (i + 1) * r * 0.15 * (1 + s));
                    g.lineStyle(0);
                }
            }
            g.beginFill(0xffffff, 0.95);
            g.drawCircle(0, -r * 0.05, r * 0.1);
            g.endFill();
        } else if (idx === 83) {
            // Pond: пруд — эллипс воды с бегущей рябью и островком-точкой (Point).
            g.beginFill(col, 0.18);
            g.drawEllipse(0, 0, r * 1.25, r * 0.85);
            g.endFill();
            for (let ring = 0; ring < 3; ring++) {
                const ph = (t * (0.7 + ring * 0.3) + ring * 0.4) % 1;
                g.lineStyle(2, light, (1 - ph) * 0.7);
                g.drawEllipse(ph * r * 0.15, 0, r * (0.2 + ph * 0.85 - ring * 0.1), r * (0.14 + ph * 0.55 - ring * 0.07));
                g.lineStyle(0);
            }
            g.lineStyle(2.5, 0x000000, 0.5);
            g.beginFill(dark, 1);
            g.drawCircle(0, r * 0.1, r * 0.3);
            g.endFill();
            g.beginFill(col, 1);
            g.drawCircle(0, r * 0.1, r * 0.26);
            g.endFill();
            g.beginFill(0xffffff, 0.9);
            g.drawCircle(-r * 0.06, r * 0.02, r * 0.07);
            g.endFill();
        } else if (idx === 84) {
            // Bud: бутон — лепестки поочерёдно приоткрываются, центр светится (Point).
            const open = 0.5 + 0.5 * Math.sin(t * 1.6);
            g.beginFill(col, 0.1);
            g.drawCircle(0, 0, r * 1.3);
            g.endFill();
            for (let i = 0; i < 3; i++) {
                const a = -Math.PI / 2 + i * Math.PI * 2 / 3;
                const lean = (i % 2 === 0 ? open : 1 - open) * r * 0.38;
                g.lineStyle(2.5, 0x000000, 0.5);
                g.beginFill(i % 2 === 0 ? col : light, 1);
                g.drawEllipse(Math.cos(a) * r * 0.35 + Math.cos(a) * lean * 0.4, Math.sin(a) * r * 0.35 + Math.sin(a) * lean * 0.4, r * 0.3, r * 0.34);
                g.endFill();
            }
            g.beginFill(0xffe08a, 0.6 + 0.4 * Math.sin(t * 4));
            g.drawCircle(0, r * 0.05, r * 0.12);
            g.endFill();
        } else if (idx === 85) {
            // Vine: вьющаяся струна-синусоида, по ней бежит светящийся узелок (Point).
            g.beginFill(col, 0.1);
            g.drawCircle(0, 0, r * 1.3);
            g.endFill();
            g.lineStyle(3.5, col, 1);
            g.moveTo(-r * 0.95, Math.sin(t * 1.2) * r * 0.3);
            for (let i = 1; i <= 16; i++) {
                const fx = -r * 0.95 + (i / 16) * r * 1.9;
                g.lineTo(fx, Math.sin(i * 0.9 + t * 1.4) * r * 0.34);
            }
            g.lineStyle(0);
            const wp = (t * 0.8) % 1;
            const wx = -r * 0.95 + wp * r * 1.9;
            const wy = Math.sin(wp * 14.4 + t * 1.4) * r * 0.34;
            g.beginFill(0xffffff, 0.95);
            g.drawCircle(wx, wy, r * 0.11 * (0.85 + 0.3 * Math.sin(t * 6)));
            g.endFill();
        } else if (idx === 86) {
            // Lily: кувшинка — 4 лепестка, лёгкое вращение, капля на листе (Point).
            g.beginFill(col, 0.12);
            g.drawCircle(0, 0, r * 1.3);
            g.endFill();
            const rot = t * 0.35;
            for (let i = 0; i < 4; i++) {
                const a = rot + i * Math.PI / 2;
                g.lineStyle(2.5, 0x000000, 0.5);
                g.beginFill(i % 2 === 0 ? col : light, 1);
                g.drawEllipse(Math.cos(a) * r * 0.32, Math.sin(a) * r * 0.32, r * 0.22, r * 0.4);
                g.endFill();
            }
            g.lineStyle(0);
            g.beginFill(0xffe08a, 0.9);
            g.drawCircle(0, 0, r * 0.16);
            g.endFill();
            const dropA = (Math.sin(t * 1.3) + 1) / 2;
            g.beginFill(0xa7f3d0, 0.75);
            g.drawCircle(-r * 0.3 + dropA * r * 0.28, r * 0.34, r * 0.07);
            g.endFill();
        } else if (idx === 87) {
            // Moss: комок мха с колышущимися ворсинками и спорами (Point).
            const wave = t * 2.4;
            g.beginFill(dark, 0.3);
            g.drawCircle(0, r * 0.15, r * 1.0);
            g.endFill();
            g.beginFill(col, 0.25);
            g.drawCircle(0, r * 0.05, r * 0.92);
            g.endFill();
            for (let i = 0; i < 12; i++) {
                const a = (i / 12) * Math.PI * 2;
                const sway = Math.sin(wave + i) * r * 0.1;
                const bx = Math.cos(a) * r * 0.8;
                const by = Math.sin(a) * r * 0.8;
                g.lineStyle(2, i % 3 === 0 ? light : col, 0.9);
                g.moveTo(bx, by);
                g.lineTo(bx + Math.cos(a) * r * 0.22 + sway * 0.5, by + Math.sin(a) * r * 0.22 - r * 0.08);
                g.lineStyle(0);
            }
            g.beginFill(0xffffff, 0.85);
            g.drawCircle(0, r * 0.05, r * 0.1);
            g.endFill();
        } else if (idx === 88) {
            // Bulb: луковица — набалдашник с корешками, росток тянется вверх (Point).
            const stretch = 0.5 + 0.5 * Math.sin(t * 1.8);
            g.beginFill(dark, 0.3);
            g.drawCircle(0, r * 0.28, r * 0.55);
            g.endFill();
            g.lineStyle(2, dark, 0.9);
            for (let i = -2; i <= 2; i++) {
                g.moveTo(i * r * 0.16, r * 0.8);
                g.lineTo(i * r * 0.22, r * 0.38);
                g.lineTo(i * r * 0.3, r * 0.8);
            }
            g.lineStyle(2.5, 0x000000, 0.55);
            g.beginFill(col, 1);
            g.drawCircle(0, r * 0.1, r * 0.5);
            g.endFill();
            g.lineStyle(2.5, light, 1);
            g.moveTo(0, -r * 0.15);
            g.lineTo(0, -r * (0.45 + stretch * 0.4));
            g.lineStyle(0);
            g.beginFill(0xffffff, 0.85 + 0.15 * Math.sin(t * 5));
            g.drawCircle(0, -r * (0.45 + stretch * 0.4), r * 0.09);
            g.endFill();
        } else if (idx === 89) {
            // Sprig: веточка с тремя листиками, наклоняющимися по очереди (Point).
            g.beginFill(col, 0.1);
            g.drawCircle(0, 0, r * 1.3);
            g.endFill();
            g.lineStyle(3, dark, 1);
            g.moveTo(0, r * 0.72);
            g.lineTo(0, -r * 0.55);
            g.lineStyle(0);
            const active = Math.floor(t * 1.2) % 3;
            for (let i = 0; i < 3; i++) {
                const tilt = (i === active) ? 0.75 : 0.35;
                const ty = r * 0.45 - i * r * 0.38;
                const x = Math.cos(tilt) * r * 0.42;
                const y = ty - Math.sin(tilt) * r * 0.3;
                g.lineStyle(2, dark, 1);
                g.moveTo(0, ty);
                g.lineTo(x, y);
                g.lineStyle(0);
                g.beginFill(i % 2 === 0 ? col : light, 1);
                g.drawEllipse(x, y, r * 0.22, r * 0.13);
                g.endFill();
            }
        } else if (idx === 90) {
            // Vein: полупрозрачная капля с пульсирующей жилкой внутри (Point).
            const pulse = 0.5 + 0.5 * Math.sin(t * 2.6);
            g.beginFill(col, 0.2 + 0.1 * pulse);
            g.drawEllipse(0, 0, r * 0.75, r * 0.95);
            g.endFill();
            g.lineStyle(2.5, 0x000000, 0.55);
            g.beginFill(col, 1);
            g.moveTo(r * 0.3, r * 0.0);
            g.lineTo(r * 0.0, -r * 0.95);
            g.lineTo(-r * 0.3, r * 0.0);
            g.lineTo(0, r * 0.55);
            g.closePath();
            g.endFill();
            g.beginFill(dark, 0.4);
            g.moveTo(r * 0.3, r * 0.0);
            g.lineTo(r * 0.0, -r * 0.95);
            g.lineTo(-r * 0.3, r * 0.0);
            g.lineTo(0, r * 0.2);
            g.closePath();
            g.endFill();
            g.lineStyle(2.5, 0xffffff, 0.4 + 0.6 * pulse);
            g.moveTo(r * 0.14, 0);
            g.lineTo(0, -r * 0.55);
            g.lineTo(-r * 0.12, r * 0.12);
            g.lineStyle(0);
        } else if (idx === 91) {
            // Grove: маленькая роща — три стволика с растущими макушками (Point).
            const ph = (t * 0.3) % 1;
            g.beginFill(col, 0.1);
            g.drawCircle(0, 0, r * 1.35);
            g.endFill();
            for (let i = -1; i <= 1; i++) {
                const x = i * r * 0.42;
                g.lineStyle(2.5, dark, 1);
                g.moveTo(x, r * 0.8);
                g.lineTo(x, -r * 0.2);
                g.lineStyle(0);
                const cr = r * (0.26 + 0.06 * Math.sin(t * 2.5 + i));
                g.beginFill(i === 0 ? col : light, 1);
                g.drawCircle(x, -r * 0.32, cr);
                g.endFill();
                g.beginFill(0xffffff, 0.5);
                g.drawCircle(x - cr * 0.3, -r * 0.32 - cr * 0.3, cr * 0.3);
                g.endFill();
            }
            g.beginFill(0xffffff, (1 - ph) * 0.6);
            g.drawCircle(-(0.5 - ph) * r * 1.3, -r * 0.6, r * 0.08);
            g.endFill();
        } else if (idx === 92) {
            // Nest: гнездо из веточек с тремя яйцами, покачивается (Point).
            const sway = Math.sin(t * 1.4) * r * 0.05;
            g.beginFill(col, 0.1);
            g.drawCircle(0, 0, r * 1.3);
            g.endFill();
            g.lineStyle(3.5, dark, 1);
            g.moveTo(-r * 0.8, sway + r * 0.05);
            for (let i = 1; i <= 8; i++) {
                const fx = -r * 0.8 + (i / 8) * r * 1.6;
                g.lineTo(fx, sway + r * (0.05 + (i % 2 === 0 ? 0.16 : -0.05)));
            }
            g.lineStyle(0);
            g.beginFill(dark, 0.35);
            g.drawEllipse(0, sway, r * 0.55, r * 0.32);
            g.endFill();
            for (let i = 0; i < 3; i++) {
                const e = i === (Math.floor(t * 2) % 3);
                g.beginFill(e ? 0xffffff : light, 1);
                g.drawEllipse(-r * 0.22 + i * r * 0.22, sway - r * 0.06, r * 0.16, r * 0.2);
                g.endFill();
            }
        } else if (idx === 93) {
            // Sap: капля сока — внутри перетекающий пузырёк, «сочится» (Point).
            const dr = 0.5 + 0.5 * Math.sin(t * 2.2);
            g.beginFill(dark, 0.25);
            g.drawCircle(0, r * 0.12, r * 1.0);
            g.endFill();
            g.lineStyle(2.5, 0x000000, 0.55);
            g.beginFill(col, 1);
            g.drawCircle(0, 0, r * 0.85);
            g.endFill();
            g.beginFill(dark, 0.4);
            g.drawCircle(0, -r * 0.12, r * 0.85);
            g.endFill();
            g.beginFill(0xffffff, 0.5 + 0.3 * dr);
            g.drawCircle(Math.sin(t * 1.5) * r * 0.18, -r * 0.18, r * 0.16 * (0.7 + 0.5 * dr));
            g.endFill();
            g.lineStyle(0);
            g.beginFill(0xffe08a, 0.7 + 0.3 * Math.sin(t * 5));
            g.drawCircle(0, r * 0.6, r * 0.08);
            g.endFill();
        } else if (idx === 94) {
            // Breeze: две струи ветра — точки-порывы бегут по пересекающимся линиям (Point).
            g.beginFill(col, 0.1);
            g.drawCircle(0, 0, r * 1.3);
            g.endFill();
            for (let s = 0; s < 2; s++) {
                g.lineStyle(2.5, s === 0 ? col : light, 0.7);
                g.moveTo(-r * 0.95, s === 0 ? -r * 0.1 : r * 0.1);
                g.lineTo(r * 0.95, s === 0 ? r * 0.1 : -r * 0.1);
                g.lineStyle(0);
                const p = ((t * (s === 0 ? 1.2 : 0.9) + s * 0.5) % 1) * 1.9 - 0.95;
                const py = (p * (s === 0 ? 1 : -1)) * r * 0.2;
                const yy = (s === 0 ? -r * 0.1 : r * 0.1) + py;
                g.beginFill(0xffffff, 0.9);
                g.drawCircle(p * r, yy, r * 0.1);
                g.endFill();
            }
        } else if (idx === 95) {
            // Haven: арка-убежище из растительности, внутри мигает огонёк (Point).
            const glow = 0.5 + 0.5 * Math.sin(t * 3);
            g.beginFill(col, 0.1);
            g.drawCircle(0, 0, r * 1.3);
            g.endFill();
            g.lineStyle(3, dark, 1);
            g.moveTo(-r * 0.7, r * 0.6);
            g.lineTo(-r * 0.7, -r * 0.1);
            g.lineTo(0, -r * 0.8);
            g.lineTo(r * 0.7, -r * 0.1);
            g.lineTo(r * 0.7, r * 0.6);
            g.lineStyle(0);
            g.beginFill(dark, 0.25);
            g.moveTo(-r * 0.7, r * 0.6);
            g.lineTo(-r * 0.7, -r * 0.1);
            g.lineTo(0, -r * 0.8);
            g.lineTo(r * 0.7, -r * 0.1);
            g.lineTo(r * 0.7, r * 0.6);
            g.closePath();
            g.endFill();
            g.beginFill(0xffc078, 0.35 + 0.65 * glow);
            g.drawCircle(0, 0, r * 0.14);
            g.endFill();
            g.lineStyle(2, light, 0.8);
            g.drawCircle(0, 0, r * 0.24);
            g.lineStyle(0);
        } else if (idx === 96) {
            // Beacon: маяк — башня с вращающимся лучом и мигающим огнём (Pentagon).
            const rot = t * 1.1;
            g.beginFill(col, 0.1);
            g.drawCircle(0, 0, r * 1.35);
            g.endFill();
            g.lineStyle(5, 0xffe08a, 0.25);
            g.moveTo(0, -r * 0.1);
            g.lineTo(Math.cos(rot) * r * 1.35, -r * 0.1 + Math.sin(rot) * r * 1.35);
            g.lineStyle(0);
            g.lineStyle(3, 0x000000, 0.55);
            g.beginFill(dark, 1);
            g.moveTo(-r * 0.22, r * 0.85);
            g.lineTo(-r * 0.16, 0);
            g.lineTo(r * 0.16, 0);
            g.lineTo(r * 0.22, r * 0.85);
            g.closePath();
            g.endFill();
            g.lineStyle(2.5, light, 1);
            g.moveTo(-r * 0.16, 0);
            g.lineTo(r * 0.16, 0);
            g.lineStyle(0);
            g.beginFill(light, 1);
            g.drawCircle(-r * 0.02, r * 0.45, r * 0.1);
            g.drawCircle(r * 0.03, r * 0.65, r * 0.08);
            g.endFill();
            g.beginFill(0xffe08a, 0.35 + 0.65 * Math.sin(t * 5));
            g.drawCircle(0, -r * 0.12, r * 0.1);
            g.endFill();
        } else if (idx === 97) {
            // Halo: пятиугольник с «нимбом» — по кольцу обращается искра (Pentagon).
            const aa = t * 0.9;
            g.beginFill(col, 0.12);
            g.drawCircle(0, 0, r * 1.3);
            g.endFill();
            g.lineStyle(3, 0x000000, 0.55);
            g.beginFill(col, 1);
            g.moveTo(Math.cos(-Math.PI / 2) * r * 0.66, Math.sin(-Math.PI / 2) * r * 0.66);
            for (let i = 1; i <= 5; i++) {
                const a = -Math.PI / 2 + i * Math.PI * 2 / 5;
                g.lineTo(Math.cos(a) * r * 0.66, Math.sin(a) * r * 0.66);
            }
            g.closePath();
            g.endFill();
            g.lineStyle(2.5, light, 0.5);
            g.drawCircle(0, 0, r * 0.8);
            g.lineStyle(0);
            g.beginFill(0xffffff, 0.95);
            g.drawCircle(Math.cos(aa) * r * 0.8, Math.sin(aa) * r * 0.8, r * 0.09);
            g.endFill();
            g.beginFill(0xffffff, 0.7);
            g.drawCircle(0, 0, r * 0.1);
            g.endFill();
        } else if (idx === 98) {
            // Crest: рельефный гребень — волнообразные слои внутри пятиугольника (Pentagon).
            g.beginFill(col, 0.12);
            g.drawCircle(0, 0, r * 1.3);
            g.endFill();
            g.lineStyle(3, 0x000000, 0.55);
            g.beginFill(dark, 1);
            g.moveTo(0, -r * 0.8);
            for (let i = 1; i < 5; i++) {
                const a = -Math.PI / 2 + i * Math.PI * 2 / 5;
                g.lineTo(Math.cos(a) * r * 0.7, Math.sin(a) * r * 0.7);
            }
            g.closePath();
            g.endFill();
            g.lineStyle(2.5, light, 0.85);
            for (let i = 0; i < 3; i++) {
                const off = (t * 0.8 + i * 0.4) % 1;
                const wy = -r * 0.2 + i * r * 0.25 + Math.sin(off * Math.PI * 2) * r * 0.12;
                g.moveTo(-r * 0.55, wy);
                g.lineTo(r * 0.55, wy);
            }
            g.lineStyle(0);
            g.beginFill(0xffffff, 0.8);
            g.drawCircle(0, -r * 0.5, r * 0.07);
            g.endFill();
        } else if (idx === 99) {
            // Lantern: фонарь — шестигранный корпус с мерцающим внутренним светом (Pentagon).
            const glow = 0.5 + 0.5 * Math.sin(t * 4);
            g.beginFill(col, 0.12);
            g.drawCircle(0, 0, r * 1.35);
            g.endFill();
            g.lineStyle(3, 0x000000, 0.55);
            g.beginFill(dark, 1);
            g.moveTo(-r * 0.32, r * 0.55);
            g.lineTo(r * 0.32, r * 0.55);
            g.lineTo(r * 0.46, 0);
            g.lineTo(r * 0.32, -r * 0.55);
            g.lineTo(-r * 0.32, -r * 0.55);
            g.lineTo(-r * 0.46, 0);
            g.closePath();
            g.endFill();
            g.beginFill(0xffe08a, 0.3 + 0.7 * glow);
            g.drawCircle(0, 0, r * 0.28);
            g.endFill();
            g.lineStyle(2, 0xffffff, 0.6);
            g.drawCircle(0, 0, r * 0.28);
            g.lineStyle(2.5, light, 1);
            g.moveTo(-r * 0.18, r * 0.55);
            g.lineTo(r * 0.18, r * 0.55);
            g.lineStyle(0);
        } else if (idx === 100) {
            // Spire: шпиль — вытянутый кристалл вверх, по ребру бежит искра (Pentagon).
            g.beginFill(col, 0.1);
            g.drawCircle(0, 0, r * 1.35);
            g.endFill();
            g.lineStyle(3, 0x000000, 0.55);
            g.beginFill(col, 1);
            g.moveTo(0, -r * 0.95);
            g.lineTo(r * 0.4, -r * 0.1);
            g.lineTo(r * 0.2, r * 0.8);
            g.lineTo(-r * 0.2, r * 0.8);
            g.lineTo(-r * 0.4, -r * 0.1);
            g.closePath();
            g.endFill();
            g.beginFill(light, 1);
            g.moveTo(0, -r * 0.95);
            g.lineTo(r * 0.4, -r * 0.1);
            g.lineTo(0, 0);
            g.closePath();
            g.endFill();
            const ph = (t * 0.7) % 1;
            const sy = -r * 0.95 + ph * r * 1.75;
            g.beginFill(0xffffff, 0.8);
            g.drawEllipse(0, sy, r * 0.07, r * 0.12);
            g.endFill();
        } else if (idx === 101) {
            // Aurora: три сияющие волновые ленты (синусоиды) (Pentagon).
            g.beginFill(col, 0.1);
            g.drawCircle(0, 0, r * 1.35);
            g.endFill();
            const bands = [col, light, 0xffffff];
            for (let b = 0; b < 3; b++) {
                g.lineStyle(3.5, bands[b], 0.25 + 0.2 * Math.sin(t * 2 + b * 2));
                g.moveTo(-r * 1.0, (b - 1) * r * 0.34 + Math.sin(t * 1.1 + b * 2) * r * 0.12);
                for (let i = 1; i <= 12; i++) {
                    const fx = -r * 1.0 + (i / 12) * r * 2.0;
                    const fy = (b - 1) * r * 0.34 + Math.sin(fx * 0.9 + t * 1.5 + b * 1.7) * r * 0.16;
                    g.lineTo(fx, fy);
                }
                g.lineStyle(0);
            }
            g.beginFill(0xffffff, 0.85);
            g.drawCircle(0, r * 0.4, r * 0.08);
            g.endFill();
        } else if (idx === 102) {
            // Prism: прозрачная призма — внутри блуждает луч преломления (Pentagon).
            g.beginFill(col, 0.15);
            g.drawCircle(0, 0, r * 1.3);
            g.endFill();
            g.lineStyle(3, 0x000000, 0.55);
            g.beginFill(col, 0.7);
            g.moveTo(0, -r * 0.85);
            g.lineTo(r * 0.85, r * 0.4);
            g.lineTo(-r * 0.85, r * 0.4);
            g.closePath();
            g.endFill();
            const ph = ((t * 0.6) % 1) * 2 - 1;
            const by = -r * 0.6 + Math.abs(ph) * r * 1.1;
            g.lineStyle(2.5, 0xffffff, 0.7);
            g.moveTo(ph * r * 0.4, -r * 0.7);
            g.lineTo(ph * r * 0.55, by);
            g.lineTo(-ph * r * 0.2, r * 0.4);
            g.lineStyle(0);
            g.beginFill(0xffffff, 0.9);
            g.drawCircle(-ph * r * 0.2, r * 0.4, r * 0.08);
            g.endFill();
        } else if (idx === 103) {
            // Ripple: пятиугольная плитка с расходящимися волнами (Pentagon).
            g.beginFill(col, 0.12);
            g.drawCircle(0, 0, r * 1.35);
            g.endFill();
            for (let ring = 0; ring < 3; ring++) {
                const ph = ((t * 0.6 + ring * 0.3) % 1);
                const rr = r * (0.25 + ph * 0.6);
                g.lineStyle(2, ring === 1 ? light : col, (1 - ph) * 0.8);
                g.moveTo(Math.cos(-Math.PI / 2) * rr, Math.sin(-Math.PI / 2) * rr);
                for (let i = 1; i <= 5; i++) {
                    const a = -Math.PI / 2 + i * Math.PI * 2 / 5;
                    g.lineTo(Math.cos(a) * rr, Math.sin(a) * rr);
                }
                g.closePath();
                g.lineStyle(0);
            }
            g.lineStyle(3, 0x000000, 0.55);
            g.beginFill(dark, 1);
            g.drawCircle(0, 0, r * 0.22);
            g.endFill();
        } else if (idx === 104) {
            // Nexus: узел с пятью «дышащими» лучами (Pentagon).
            g.beginFill(col, 0.1);
            g.drawCircle(0, 0, r * 1.35);
            g.endFill();
            for (let i = 0; i < 5; i++) {
                const a = -Math.PI / 2 + i * Math.PI * 2 / 5;
                const len = r * (0.5 + 0.22 * Math.sin(t * 2.2 + i * 1.3));
                g.lineStyle(2.5, i % 2 === 0 ? light : col, 0.9);
                g.moveTo(Math.cos(a) * r * 0.18, Math.sin(a) * r * 0.18);
                g.lineTo(Math.cos(a) * len, Math.sin(a) * len);
                g.lineStyle(0);
                g.beginFill(0xffffff, 0.8);
                g.drawCircle(Math.cos(a) * len, Math.sin(a) * len, r * 0.07);
                g.endFill();
            }
            g.lineStyle(3, 0x000000, 0.55);
            g.beginFill(col, 1);
            g.drawCircle(0, 0, r * 0.22);
            g.endFill();
            g.beginFill(0xffffff, 0.9);
            g.drawCircle(0, 0, r * 0.08);
            g.endFill();
        } else if (idx === 105) {
            // Diamond: алмаз с гранями, блик бегает внутри (Pentagon).
            g.beginFill(col, 0.12);
            g.drawCircle(0, 0, r * 1.35);
            g.endFill();
            g.lineStyle(3, 0x000000, 0.55);
            g.beginFill(col, 1);
            g.moveTo(0, -r * 0.85);
            g.lineTo(r * 0.62, 0);
            g.lineTo(0, r * 0.85);
            g.lineTo(-r * 0.62, 0);
            g.closePath();
            g.endFill();
            g.lineStyle(2.5, light, 0.7);
            g.moveTo(0, -r * 0.85);
            g.lineTo(0, r * 0.85);
            g.moveTo(-r * 0.62, 0);
            g.lineTo(r * 0.62, 0);
            g.lineStyle(0);
            const sa = t * 1.4;
            g.beginFill(0xffffff, 0.55 + 0.35 * Math.sin(t * 4));
            g.drawCircle(Math.max(-r * 0.3, Math.min(r * 0.3, Math.cos(sa) * r * 0.5)), Math.max(-r * 0.4, Math.min(r * 0.4, Math.sin(sa) * r * 0.4)), r * 0.14);
            g.endFill();
        } else if (idx === 106) {
            // Obelisk: обелиск — колонна, световая полоса идёт вверх (Pentagon).
            g.beginFill(col, 0.12);
            g.drawCircle(0, 0, r * 1.35);
            g.endFill();
            g.lineStyle(3, 0x000000, 0.55);
            g.beginFill(dark, 1);
            g.moveTo(-r * 0.34, r * 0.85);
            g.lineTo(-r * 0.24, -r * 0.1);
            g.lineTo(0, -r * 0.8);
            g.lineTo(r * 0.24, -r * 0.1);
            g.lineTo(r * 0.34, r * 0.85);
            g.closePath();
            g.endFill();
            g.beginFill(col, 1);
            g.moveTo(-r * 0.24, -r * 0.1);
            g.lineTo(0, -r * 0.8);
            g.lineTo(r * 0.24, -r * 0.1);
            g.closePath();
            g.endFill();
            const uph = (t * 0.5) % 1;
            const uy = -r * 0.8 + uph * r * 1.5;
            g.beginFill(0xffffff, 0.4 + 0.4 * Math.sin(t * 3));
            g.drawEllipse(0, uy, r * 0.2, r * 0.16);
            g.endFill();
        } else if (idx === 107) {
            // Lens: линза — кольцо фокуса сжимается, блик наводится (Pentagon).
            const foc = 0.5 + 0.5 * Math.sin(t * 1.6);
            g.beginFill(col, 0.12);
            g.drawCircle(0, 0, r * 1.35);
            g.endFill();
            g.lineStyle(3, 0x000000, 0.55);
            g.beginFill(col, 0.85);
            g.drawCircle(0, 0, r * 0.75);
            g.endFill();
            g.beginFill(0xffffff, 0.12);
            g.drawCircle(-r * 0.2, -r * 0.22, r * 0.7);
            g.endFill();
            g.lineStyle(2.5, light, 0.6);
            g.drawCircle(0, 0, r * (0.25 + foc * 0.5));
            g.lineStyle(0);
            g.beginFill(0xffffff, 0.95);
            g.drawCircle(-r * 0.18, -r * 0.2, r * 0.1);
            g.endFill();
        } else if (idx === 108) {
            // Pylon: пилон — столб с перекладинами и бегущими индикаторами (Pentagon).
            g.beginFill(col, 0.12);
            g.drawCircle(0, 0, r * 1.35);
            g.endFill();
            g.lineStyle(4, dark, 1);
            g.moveTo(0, r * 0.85);
            g.lineTo(0, -r * 0.7);
            g.lineStyle(0);
            for (let i = 0; i < 3; i++) {
                const y = r * 0.35 - i * r * 0.38;
                const w = r * (0.66 - i * 0.12);
                g.lineStyle(3, i % 2 === 0 ? col : light, 1);
                g.moveTo(-w, y);
                g.lineTo(w, y);
                g.lineStyle(0);
                const p = (t * 1.1 + i * 0.4) % 1;
                g.beginFill(0xffffff, 0.9);
                g.drawCircle(-w + p * w * 2, y - r * 0.06, r * 0.07);
                g.endFill();
            }
            g.beginFill(0xffffff, 0.7 + 0.3 * Math.sin(t * 5));
            g.drawCircle(0, -r * 0.7, r * 0.09);
            g.endFill();
        } else if (idx === 109) {
            // Monolith: тёмный монолит с тлеющей щелью (Pentagon).
            const glow = 0.5 + 0.5 * Math.sin(t * 2.4);
            g.beginFill(col, 0.1);
            g.drawCircle(0, 0, r * 1.35);
            g.endFill();
            g.lineStyle(3, 0x000000, 0.6);
            g.beginFill(dark, 1);
            g.moveTo(-r * 0.5, r * 0.85);
            g.lineTo(-r * 0.36, -r * 0.7);
            g.lineTo(r * 0.36, -r * 0.7);
            g.lineTo(r * 0.5, r * 0.85);
            g.closePath();
            g.endFill();
            g.beginFill(0xff8a65, 0.25 + 0.75 * glow);
            g.moveTo(-r * 0.16, -r * 0.55);
            g.lineTo(-r * 0.05, r * 0.7);
            g.lineTo(r * 0.05, r * 0.7);
            g.lineTo(r * 0.16, -r * 0.55);
            g.closePath();
            g.endFill();
            g.beginFill(0xffffff, 0.6);
            g.drawCircle(0, -r * 0.55, r * 0.06);
            g.endFill();
        } else if (idx === 110) {
            // Corona: корона — пятиугольник с «лампами» по обводу, светятся по кругу (Pentagon).
            g.beginFill(col, 0.1);
            g.drawCircle(0, 0, r * 1.35);
            g.endFill();
            g.lineStyle(3, 0x000000, 0.55);
            g.beginFill(dark, 1);
            g.moveTo(0, -r * 0.75);
            for (let i = 1; i < 5; i++) {
                const a = -Math.PI / 2 + i * Math.PI * 2 / 5;
                g.lineTo(Math.cos(a) * r * 0.6, Math.sin(a) * r * 0.6);
            }
            g.closePath();
            g.endFill();
            for (let i = 0; i < 5; i++) {
                const a = -Math.PI / 2 + i * Math.PI * 2 / 5;
                const lit = Math.sin(t * 2.2 + i * 1.2) > 0;
                g.beginFill(lit ? 0xffe08a : 0xffffff, lit ? 0.95 : 0.4);
                g.drawCircle(Math.cos(a) * r * 0.6, Math.sin(a) * r * 0.6, r * 0.1);
                g.endFill();
            }
            g.beginFill(0xffffff, 0.75);
            g.drawCircle(0, 0, r * 0.12);
            g.endFill();
        } else if (idx === 111) {
            // Trinity: единица тройного «организма» — одиночный кружок с мягким
            // «дыханием» и бликом (реальный моб; витрина показывает троих вместе).
            const gg = 0.5 + 0.5 * Math.sin(t * 2.6);
            g.beginFill(dark, 0.2);
            g.drawCircle(0, 0, r * 1.25);
            g.endFill();
            g.lineStyle(2.5, 0x000000, 0.55);
            g.beginFill(col, 1);
            g.drawCircle(0, 0, r * (0.8 + 0.1 * gg));
            g.endFill();
            g.lineStyle(0);
            g.beginFill(dark, 0.35);
            g.drawCircle(0, -r * 0.12 * (1 + 0.2 * gg), r * (0.8 + 0.1 * gg));
            g.endFill();
            g.beginFill(0xffffff, 0.35 + 0.45 * gg);
            g.drawCircle(-r * 0.16, -r * 0.2, r * 0.14);
            g.endFill();
            g.beginFill(light, 0.7);
            g.drawCircle(r * 0.12, r * 0.2, r * 0.06);
            g.endFill();
        }
    else {
        g.beginFill(col, 0.9);
        g.drawCircle(0, 0, r * 0.8);
        g.endFill();
        g.lineStyle(2.5, light, 1);
        g.drawCircle(0, 0, r * 0.8);
        g.lineStyle(0);
    }
}
