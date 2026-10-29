// wave_decor.js — фон меню WAVEBORN: «волны-осциллографы».
// Несколько неоновых синусоид (циан/маджента/лайм) плывут по экрану,
// поверх — редкие глитч-вспышки и мелкие звёзды-точки. Vanilla 2D-canvas:
// без PIXI, один requestAnimationFrame, дешёвая перерисовка.
//
// API: startWaveDecor() / stopWaveDecor() — вызываются из showScreen (main.js).
// Тумблер глитч-эффектов (настройки WAVEBORN) читает window.WB_GLI === false.

(function () {
    'use strict';

    let canvas = null, ctx = null, raf = 0;
    let running = false;
    let W = 0, H = 0, t = 0;
    let stars = [];

    // Палитра WAVEBORN (неон-аркада)
    const CYAN = '0, 240, 255';
    const MAGENTA = '255, 46, 136';
    const LIME = '182, 255, 46';

    // Каждая волна: своя амплитуда/длина/скорость/цвет/толщина
    const waves = [
        { amp: 46, len: 0.0038, speed: 0.9,  y: 0.30, col: CYAN,    lw: 2.0, alpha: 0.55 },
        { amp: 30, len: 0.0062, speed: -1.3, y: 0.44, col: MAGENTA, lw: 1.6, alpha: 0.40 },
        { amp: 62, len: 0.0026, speed: 0.55, y: 0.60, col: CYAN,    lw: 1.2, alpha: 0.28 },
        { amp: 22, len: 0.0090, speed: 1.8,  y: 0.72, col: LIME,    lw: 1.3, alpha: 0.30 },
    ];

    function resize() {
        W = canvas.width = window.innerWidth;
        H = canvas.height = window.innerHeight;
        seedStars();
        seedSilhouettes();
    }

    function seedStars() {
        stars = [];
        const n = Math.floor((W * H) / 26000);
        for (let i = 0; i < n; i++) {
            stars.push({
                x: Math.random() * W,
                y: Math.random() * H,
                r: 0.6 + Math.random() * 1.4,
                a: 0.06 + Math.random() * 0.22,
                ph: Math.random() * Math.PI * 2,
            });
        }
    }

    // Плавающие силуэты мобов (RICHNESS меню): круг/квадрат/треугольник/ромб,
    // медленный дрейф, очень прозрачные — глубина сцены.
    const silhouettes = [];
    function seedSilhouettes() {
        silhouettes.length = 0;
        for (let i = 0; i < 7; i++) {
            silhouettes.push({
                x: Math.random() * W, y: Math.random() * H,
                r: 14 + Math.random() * 30,
                kind: Math.floor(Math.random() * 4), // 0 circle 1 square 2 tri 3 diamond
                vx: (Math.random() - 0.5) * 14, vy: (Math.random() - 0.5) * 10,
                a: 0.045 + Math.random() * 0.05,
                rot: Math.random() * Math.PI, vr: (Math.random() - 0.5) * 0.2,
            });
        }
    }
    function drawSilhouettes(dt) {
        for (const s0 of silhouettes) {
            s0.x += s0.vx * dt; s0.y += s0.vy * dt; s0.rot += s0.vr * dt;
            if (s0.x < -60) s0.x = W + 60; if (s0.x > W + 60) s0.x = -60;
            if (s0.y < -60) s0.y = H + 60; if (s0.y > H + 60) s0.y = -60;
            ctx.save();
            ctx.translate(s0.x, s0.y);
            ctx.rotate(s0.rot);
            ctx.strokeStyle = 'rgba(140, 200, 235,' + s0.a + ')';
            ctx.lineWidth = 2;
            ctx.beginPath();
            if (s0.kind === 0) ctx.arc(0, 0, s0.r, 0, Math.PI * 2);
            else if (s0.kind === 1) ctx.rect(-s0.r * 0.8, -s0.r * 0.8, s0.r * 1.6, s0.r * 1.6);
            else if (s0.kind === 2) {
                ctx.moveTo(0, -s0.r); ctx.lineTo(s0.r * 0.9, s0.r * 0.7); ctx.lineTo(-s0.r * 0.9, s0.r * 0.7); ctx.closePath();
            } else {
                ctx.moveTo(0, -s0.r); ctx.lineTo(s0.r * 0.7, 0); ctx.lineTo(0, s0.r); ctx.lineTo(-s0.r * 0.7, 0); ctx.closePath();
            }
            ctx.stroke();
            ctx.restore();
        }
    }

    // Глитч-вспышка: короткий горизонтальный сдвиг всей сцены (редко)
    let glitchUntil = 0, glitchOff = 0;

    function draw(dtMs) {
        t += dtMs / 1000;
        const gOn = window.WB_GLI !== false;
        if (gOn && Math.random() < 0.004 && t > glitchUntil) {
            glitchUntil = t + 0.08 + Math.random() * 0.1;
            glitchOff = (Math.random() * 14 - 7);
        } else if (t > glitchUntil) {
            glitchOff = 0;
        }

        ctx.clearRect(0, 0, W, H);

        // Силуэты мобов (за звёздами).
        drawSilhouettes(dt);

        // Звёзды-точки с мерцанием
        for (const s of stars) {
            const tw = s.a * (0.6 + 0.4 * Math.sin(t * 1.7 + s.ph));
            ctx.fillStyle = 'rgba(180, 220, 255, ' + tw.toFixed(3) + ')';
            ctx.beginPath();
            ctx.arc(s.x, s.y, s.r, 0, Math.PI * 2);
            ctx.fill();
        }

        // Осциллограф-линии: синус с двумя гармониками + «дрожание» развёртки
        for (const w of waves) {
            const baseY = H * w.y;
            ctx.beginPath();
            for (let x = -20; x <= W + 20; x += 6) {
                const ph = x * w.len + t * w.speed;
                let y = baseY
                    + Math.sin(ph) * w.amp
                    + Math.sin(ph * 2.7 + 1.3) * w.amp * 0.28
                    + Math.sin(t * 0.7 + baseY) * 3;
                // Ступенька развёртки: волна чуть «цифровая»
                y = Math.round(y / 3) * 3;
                if (x === -20) ctx.moveTo(x, y);
                else ctx.lineTo(x, y);
            }
            ctx.strokeStyle = 'rgba(' + w.col + ',' + w.alpha + ')';
            ctx.lineWidth = w.lw;
            // Неоновое свечение: вторая широкая полупрозрачная линия
            ctx.shadowColor = 'rgba(' + w.col + ',0.8)';
            ctx.shadowBlur = 12;
            ctx.stroke();
            ctx.shadowBlur = 0;
        }

        // Глитч: редкие яркие горизонтальные засветки
        if (gOn && glitchOff !== 0) {
            ctx.fillStyle = 'rgba(' + CYAN + ',0.05)';
            const gy = Math.random() * H;
            ctx.fillRect(0, gy, W, 2 + Math.random() * 3);
            ctx.save();
            ctx.translate(glitchOff, 0);
            ctx.restore();
        }
    }

    function loop(prev) {
        if (!running) return;
        const now = performance.now();
        const dt = Math.min(50, now - (prev || now));
        draw(dt);
        raf = requestAnimationFrame(loop);
    }

    function startWaveDecor() {
        canvas = document.getElementById('menuDecorCanvas');
        if (!canvas) return;
        if (!ctx) {
            ctx = canvas.getContext('2d');
            window.addEventListener('resize', () => { if (running) resize(); });
        }
        canvas.style.display = 'block';
        resize();
        if (!running) {
            running = true;
            raf = requestAnimationFrame(loop);
        }
    }

    function stopWaveDecor() {
        running = false;
        if (raf) { cancelAnimationFrame(raf); raf = 0; }
        if (canvas) canvas.style.display = 'none';
    }

    window.startWaveDecor = startWaveDecor;
    window.stopWaveDecor = stopWaveDecor;
})();
