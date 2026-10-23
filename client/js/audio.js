// audio.js — «Звук и музыка» v2.0 (NewUpdates): генеративный WebAudio-движок.
//
// Принципы (как в мокапе NewUpdates):
//   • НИ ОДНОГО файла-ассета — всё синтезируется на месте: SFX собираются из
//     осцилляторов и шумовых всплесков, музыка — процедурный секвенсор;
//   • v2.0: музыка — КОМПОЗИЦИИ, а не 4-тактовые петли. Каждая песня состоит
//     из секций (интро → тема → развитие → брейк → кульминация → разрешение)
//     с собственной плотностью, барабанами и филлами на границах;
//   • v2.0 синтез: реверб (свёртка с генерированным импульсом), стерео-ширина
//     (панорама слоёв), динамика (velocity, акценты, «человеческий» тайминг);
//   • микшер из четырёх шин: master → music / sfx / ui (Settings → Audio);
//   • звук удара петали по мобу привязан к существующему эффекту
//     window.mobHitFX (render.js) — сила/цвет удара задают тон и громкость;
//   • фоновая музыка: меню, галерея, реалм, мир, витрина Skins и редактор
//     аватара (смена настроения — по showScreen), мир — по ступени биома;
//   • ошибки здесь НИКОГДА не роняют игру: всё в try/catch, без контекста
//     (или без жеста пользователя) модуль просто молчит.
//
// API: window.GameAudio = {
//     sfx(name, opts), ui(name, opts),       // проиграть эффект
//     mood(name), currentMood(),             // музыка: menu|gallery|realm|shop|avatar|off
//     zone(),                                // трек текущей ступени биома (мир)
//     setMix({master,music,sfx,ui}), mix(),  // микшер (0..1)
//     setEnabled(bool), enabled(),
//     unlock(),                              // инициализация по жесту
// }
'use strict';

(function () {
    const AC = window.AudioContext || window.webkitAudioContext;

    // ------------------------------------------------------------ состояние
    // Значения микшера — 0..1. Дефолты: эффекты громче музыки, интерфейс тише.
    const mix = { master: 0.8, music: 0.55, sfx: 0.9, ui: 0.5 };
    let enabled = true;      // тумблер «Sound & music» в настройках
    let ctx = null;          // AudioContext (создаётся по первому жесту)
    let masterGain = null, musicGain = null, sfxGain = null, uiGain = null;
    let musicDuck = null;    // 0..1: приглушение музыки (вкладка скрыта / пауза)
    let sfxBusGain = null;   // общий «пред-усилитель» эффектов (лимитер ударов)
    let revConv = null, revGain = null; // реверб-шина музыки (свёртка)

    // ---------------------------------------------------------------- утилиты
    const now = () => (ctx ? ctx.currentTime : 0);
    const clamp = (v, a, b) => (v < a ? a : (v > b ? b : v));
    function midiHz(m) { return 440 * Math.pow(2, (m - 69) / 12); }
    function rnd(a, b) { return a + Math.random() * (b - a); }

    // ensureCtx — создать граф. Пока нет жеста пользователя контекст создаём
    // в 'suspended' — браузеры разрешают это, звук пойдёт после resume().
    function ensureCtx() {
        if (ctx || !AC || !enabled) return ctx;
        try {
            ctx = new AC();
            const comp = ctx.createDynamicsCompressor();
            comp.threshold.value = -14;
            comp.knee.value = 24;
            comp.ratio.value = 6;
            comp.attack.value = 0.004;
            comp.release.value = 0.24;

            masterGain = ctx.createGain();
            musicGain = ctx.createGain();
            sfxGain = ctx.createGain();
            uiGain = ctx.createGain();
            musicDuck = ctx.createGain();
            sfxBusGain = ctx.createGain();

            musicGain.connect(musicDuck);
            musicDuck.connect(masterGain);
            sfxBusGain.connect(sfxGain);
            sfxGain.connect(masterGain);
            uiGain.connect(masterGain);
            masterGain.connect(comp);
            comp.connect(ctx.destination);

            // Реверб музыки: свёртка с генерированным импульсом (шум с
            // экспоненциальным затуханием). Сухой сигнал трека уходит в
            // musicGain напрямую, реверб — через per-slot send (см. makeSlot).
            revConv = ctx.createConvolver();
            revConv.buffer = makeReverbBuffer();
            revGain = ctx.createGain();
            revGain.gain.value = 0.85;
            revConv.connect(revGain);
            revGain.connect(musicGain);

            applyMix();
        } catch (e) { ctx = null; }
        return ctx;
    }

    // Импульс зала: ~2.2с, затухание с небольшой «предзадержкой» (пробел в
    // начале = ощущение комнаты, а не металлического ведра).
    function makeReverbBuffer() {
        const dur = 2.2, decay = 2.6;
        const len = Math.floor(ctx.sampleRate * dur);
        const buf = ctx.createBuffer(2, len, ctx.sampleRate);
        const pre = Math.floor(ctx.sampleRate * 0.02);
        for (let ch = 0; ch < 2; ch++) {
            const d = buf.getChannelData(ch);
            for (let i = 0; i < len; i++) {
                if (i < pre) { d[i] = 0; continue; }
                const k = (i - pre) / (len - pre);
                d[i] = (Math.random() * 2 - 1) * Math.pow(1 - k, decay);
            }
        }
        return buf;
    }

    function applyMix() {
        if (!ctx) return;
        const t = now();
        const g = enabled ? 1 : 0;
        const ramp = (node, v) => {
            if (!node) return;
            node.gain.cancelScheduledValues(t);
            node.gain.setTargetAtTime(v, t, 0.05);
        };
        ramp(masterGain, (g ? clamp(mix.master, 0, 1) : 0) * 0.9);
        ramp(musicGain, clamp(mix.music, 0, 1));
        ramp(sfxGain, clamp(mix.sfx, 0, 1));
        ramp(uiGain, clamp(mix.ui, 0, 1));
    }

    // unlock — вызывается по первому жесту (клик/клавиша/тач). Браузеры не
    // дают запускать звук до жеста, поэтому и музыку, и эффекты «будим» тут.
    function unlock() {
        if (!enabled) return;
        ensureCtx();
        if (!ctx) return;
        try {
            if (ctx.state === 'suspended' && ctx.resume) ctx.resume();
        } catch (e) { /* ignore */ }
        if (pendingMood) {
            const m = pendingMood;
            pendingMood = null;
            mood(m);
        }
    }

    // ------------------------------------------------------------- примитивы
    // voice — осциллятор + ADSR-огибающая (+ опционально: фильтр с огибающей
    // среза, панорама, вибрато). Один примитив на всё: от SFX до музыки.
    function voice(freq, t0, dur, opt) {
        const o = ctx.createOscillator();
        const g = ctx.createGain();
        o.type = opt.wave || 'square';
        o.frequency.setValueAtTime(freq, t0);
        if (opt.to) o.frequency.exponentialRampToValueAtTime(Math.max(20, opt.to), t0 + dur);
        if (opt.detune) o.detune.value = opt.detune;
        if (opt.vib) {
            // Вибрато «врастает» — живой голос, а не сирена.
            const lfo = ctx.createOscillator();
            lfo.frequency.value = opt.vib;
            const lg = ctx.createGain();
            lg.gain.setValueAtTime(0, t0);
            lg.gain.linearRampToValueAtTime(opt.vibAmt || 8, t0 + Math.min(0.3, dur * 0.5));
            lfo.connect(lg);
            lg.connect(o.detune);
            lfo.start(t0);
            lfo.stop(t0 + dur + 0.05);
        }
        const atk = opt.attack != null ? opt.attack : 0.006;
        const gv = opt.gain != null ? opt.gain : 0.05;
        g.gain.setValueAtTime(0.0001, t0);
        g.gain.exponentialRampToValueAtTime(Math.max(0.0002, gv), t0 + atk);
        g.gain.exponentialRampToValueAtTime(0.0001, t0 + Math.max(atk + 0.02, dur));
        let tail = g;
        if (opt.filter) {
            const f = ctx.createBiquadFilter();
            f.type = opt.filter;
            f.frequency.setValueAtTime(opt.cutoff || 1200, t0);
            if (opt.filterTo) {
                f.frequency.exponentialRampToValueAtTime(Math.max(60, opt.filterTo), t0 + dur);
            }
            f.Q.value = opt.q || 0.8;
            g.connect(f);
            tail = f;
        }
        o.connect(g);
        let out = tail;
        if (opt.pan && ctx.createStereoPanner) {
            const p = ctx.createStereoPanner();
            p.pan.value = clamp(opt.pan, -1, 1);
            out.connect(p);
            out = p;
        }
        out.connect(opt.dest || sfxGain);
        o.start(t0);
        o.stop(t0 + dur + 0.08);
        return o;
    }

    // Шумовой всплеск (крэк/хит/шаг/тарелка): короткий буфер, обрезанный
    // фильтром. Буфер кэшируем — он одинаков для всех всплесков.
    let noiseBuf = null;
    function getNoise() {
        if (noiseBuf || !ctx) return noiseBuf;
        const len = Math.floor(ctx.sampleRate * 0.5);
        noiseBuf = ctx.createBuffer(1, len, ctx.sampleRate);
        const d = noiseBuf.getChannelData(0);
        for (let i = 0; i < len; i++) d[i] = Math.random() * 2 - 1;
        return noiseBuf;
    }
    function noiseHit(t0, dur, opt) {
        const buf = getNoise();
        if (!buf) return;
        const s = ctx.createBufferSource();
        s.buffer = buf;
        s.playbackRate.value = opt.rate || 1;
        const f = ctx.createBiquadFilter();
        f.type = opt.filter || 'bandpass';
        f.frequency.setValueAtTime(opt.freq || 1600, t0);
        if (opt.sweep) f.frequency.exponentialRampToValueAtTime(Math.max(60, opt.sweep), t0 + dur);
        f.Q.value = opt.q || 1.1;
        const g = ctx.createGain();
        const gv = opt.gain != null ? opt.gain : 0.2;
        g.gain.setValueAtTime(0.0001, t0);
        g.gain.exponentialRampToValueAtTime(gv, t0 + 0.004);
        g.gain.exponentialRampToValueAtTime(0.0001, t0 + dur);
        s.connect(f);
        f.connect(g);
        let out = g;
        if (opt.pan && ctx.createStereoPanner) {
            const p = ctx.createStereoPanner();
            p.pan.value = clamp(opt.pan, -1, 1);
            out.connect(p);
            out = p;
        }
        out.connect(opt.dest || sfxGain);
        s.start(t0);
        s.stop(t0 + dur + 0.02);
    }

    // panGain — источник со стерео-панорамой (для мировых звуков): узел
    // StereoPanner есть везде, где есть WebAudio, но на всякий случай — фолбэк.
    function busWithPan(pan, dest) {
        if (!ctx.createStereoPanner) return dest;
        const p = ctx.createStereoPanner();
        p.pan.value = clamp(pan || 0, -1, 1);
        p.connect(dest || sfxGain);
        return p;
    }

    // --------------------------------------------------------- барабаны/инструменты
    // Музыкальные «инструменты» v2.0: барабаны с телом и кликом, колокол,
    // широкий пэд, бас с сабом. Всё поверх voice()/noiseHit().

    // Бочка: питч-дроп + короткий клик наверху (а иначе кик «каша» в миксе).
    function kickAt(t, vel, dest) {
        voice(120, t, 0.16, { wave: 'sine', to: 46, gain: 0.085 * vel,
            attack: 0.002, dest: dest });
        noiseHit(t, 0.02, { freq: 3400, filter: 'highpass', q: 0.7,
            gain: 0.028 * vel, dest: dest });
    }
    // Снейр: шум + тональное «тело» (190Гц) — без тела он звучит как пшш.
    function snareAt(t, vel, lvl, dest) {
        noiseHit(t, 0.1, { freq: 1900, sweep: 900, gain: 0.05 * lvl * vel,
            q: 0.9, dest: dest });
        voice(midiHz(55), t, 0.06, { wave: 'triangle', gain: 0.026 * lvl * vel,
            dest: dest });
    }
    // Хэт: закрытый (короткий) / открытый; частота чуть плавает — «живее».
    function hatAt(t, vel, open, dest) {
        noiseHit(t, open ? 0.12 : 0.035, {
            freq: 8200 * rnd(0.94, 1.06), q: 1.4,
            gain: (open ? 0.02 : 0.015) * vel, dest: dest,
        });
    }
    // Крэш на смене секции: длинный «шшш» с медленным спадом.
    function crashAt(t, vel, dest) {
        noiseHit(t, 1.0, { freq: 5200, sweep: 1600, filter: 'highpass', q: 0.5,
            gain: 0.04 * vel, dest: dest });
        voice(midiHz(96), t, 0.7, { wave: 'sine', gain: 0.012 * vel, dest: dest });
    }
    // Колокол: основной тон + негармонические обертоны (2.76×, 4.07×) —
    // у синуса в одиночку «флейтовый» тембр, обертоны дают металл/хрусталь.
    function bellAt(m, t, dur, opt) {
        voice(midiHz(m), t, dur, { wave: 'sine', gain: opt.gain,
            attack: opt.attack || 0.004, pan: opt.pan, dest: opt.dest });
        voice(midiHz(m) * 2.76, t, dur * 0.4, { wave: 'sine',
            gain: opt.gain * 0.2, attack: 0.003, pan: opt.pan, dest: opt.dest });
        voice(midiHz(m) * 4.07, t, dur * 0.22, { wave: 'sine',
            gain: opt.gain * 0.07, attack: 0.003, dest: opt.dest });
    }
    // Пэд: на каждую ноту — пара расстроенных осцилляторов по краям стерео
    // (±0.3% частоты) + фильтр. Даёт широкое «дышащее» полотно вместо точки.
    function padAt(rootMidi, t, dur, opt) {
        (opt.ivs || [0, 7, 12, 15]).forEach((iv) => {
            const f = midiHz(rootMidi + iv);
            voice(f * 1.003, t, dur, { wave: opt.wave, gain: opt.gain * 0.55,
                attack: 0.55, pan: -0.55, filter: 'lowpass',
                cutoff: opt.cutoff || 1100, q: 0.4, dest: opt.dest });
            voice(f * 0.997, t, dur, { wave: opt.wave, gain: opt.gain * 0.55,
                attack: 0.6, pan: 0.55, filter: 'lowpass',
                cutoff: opt.cutoff || 1050, q: 0.4, dest: opt.dest });
        });
    }
    // Бас: пила с закрывающейся огибающей фильтра + саб-синус на октаву ниже.
    // Короткий dur = «палм-мьют чаггинг» (Chaos), длинный — тёплый бас.
    function bassAt(m, t, dur, opt) {
        const cut = opt.cut || 520;
        voice(midiHz(m), t, dur, { wave: opt.wave || 'sawtooth', gain: opt.gain,
            attack: 0.006, filter: 'lowpass', cutoff: cut * 1.7,
            filterTo: cut, q: 1.2, dest: opt.dest });
        voice(midiHz(m - 12), t, dur, { wave: 'sine', gain: opt.gain * 0.5,
            attack: 0.005, dest: opt.dest });
    }

    // --------------------------------------------------------------- эффекты
    // Удар петали по мобу: «крэк» + низкий «тук» + (у редких петаль) звон.
    // power 0..1 — относительный урон, ri — индекс редкости бьющей петали,
    // gain — дистанционная громкость (0..1), pan — панорама влево/вправо.
    function sfxHit(power, ri, pan, gain) {
        const t0 = now();
        const p = clamp(power != null ? power : 0.5, 0.15, 1);
        const vol = clamp(gain != null ? gain : 1, 0, 1);
        const dest = busWithPan(pan, sfxGain);
        noiseHit(t0, 0.055 + 0.05 * p, {
            freq: 1500 + p * 1200 + (ri || 0) * 120, sweep: 420, q: 1.3,
            gain: (0.1 + 0.16 * p) * vol, dest: dest,
        });
        voice(150 - p * 40, t0, 0.09 + 0.05 * p, {
            wave: 'sine', to: 62, gain: (0.09 + 0.13 * p) * vol, attack: 0.003, dest: dest,
        });
        // Редкие петали «звенят» — маленький металлический призвук.
        if ((ri || 0) >= 3) {
            voice(midiHz(84 + (ri || 0) * 2 + Math.random() * 2), t0, 0.16, {
                wave: 'triangle', gain: (0.035 + 0.01 * p) * vol, dest: dest,
            });
        }
    }

    const SFX = {
        // Удар по мобу (вызывается из хука mobHitFX; см. ниже).
        hit: (o) => sfxHit(o && o.power, o && o.ri, o && o.pan, o && o.gain),
        // Игрок получил урон: короткий низкий «вздрог».
        hurt: (o) => {
            if (!ctx) return;
            const t0 = now(), p = clamp((o && o.power) || 0.4, 0.15, 1);
            voice(320 - 60 * p, t0, 0.16, { wave: 'sawtooth', to: 96, gain: 0.07 + 0.06 * p });
            noiseHit(t0, 0.08, { freq: 700, sweep: 240, gain: 0.05 + 0.06 * p });
        },
        // Петаль сломалась / моб убит — «хлопок».
        break: () => {
            if (!ctx) return;
            const t0 = now();
            noiseHit(t0, 0.14, { freq: 2600, sweep: 380, gain: 0.14, q: 0.9 });
            voice(220, t0 + 0.01, 0.12, { wave: 'triangle', to: 90, gain: 0.05 });
        },
        // МОЁ убийство моба (сервер шлёт youKilled только убийце): сочный «поп»
        // — глухой тук + крэк + наградной чирп вверх. ri моба масштабирует звон.
        mobKill: (o) => {
            if (!ctx) return;
            const t0 = now();
            const ri = clamp((o && o.ri) || 0, 0, 15);
            voice(190 - ri * 4, t0, 0.14, { wave: 'sine', to: 55, gain: 0.14,
                attack: 0.002 });
            noiseHit(t0, 0.1, { freq: 2400 + ri * 90, sweep: 420, gain: 0.1, q: 0.9 });
            // Награда: две ноты вверх (терция), у глубоких мобов — третья.
            const base = 76 + Math.min(4, ri);
            voice(midiHz(base), t0 + 0.07, 0.1, { wave: 'triangle', gain: 0.05 });
            voice(midiHz(base + 4), t0 + 0.14, 0.14, { wave: 'triangle', gain: 0.05 });
            if (ri >= 5) {
                voice(midiHz(base + 9), t0 + 0.21, 0.2, { wave: 'triangle', gain: 0.045 });
                noiseHit(t0 + 0.07, 0.3, { freq: 5200, sweep: 2600, q: 0.7, gain: 0.03 });
            }
        },
        // Редкий дроп на земле (Hexagon+): сверкающий sting — каскад колоколов
        // вверх + шумовой «блеск». Чем глубже редкость, тем длиннее и ярче.
        rareDrop: (o) => {
            if (!ctx) return;
            const ri = clamp((o && o.ri) != null ? (o && o.ri) : 5, 5, 15);
            const t0 = now();
            const deep = ri - 5;                 // 0..10
            noiseHit(t0, 0.5 + deep * 0.06, { freq: 600, sweep: 6800, q: 0.7,
                gain: 0.05 + deep * 0.004 });
            const notes = [0, 7, 12, 16, 19].slice(0, 3 + Math.min(2, deep));
            notes.forEach((s, i) => {
                const m = 86 + ri + s;
                bellAt(m, t0 + 0.08 + i * 0.09, 0.7 + deep * 0.08,
                    { gain: 0.045 + deep * 0.003 });
            });
        },
        // Подбор предмета/осколка: две ноты вверх.
        pickup: () => {
            if (!ctx) return;
            const t0 = now();
            voice(midiHz(76), t0, 0.1, { wave: 'square', gain: 0.05 });
            voice(midiHz(83), t0 + 0.07, 0.12, { wave: 'square', gain: 0.045 });
        },
        // Портал/смена зоны: шумовой «свуп» + подъём.
        portal: () => {
            if (!ctx) return;
            const t0 = now();
            noiseHit(t0, 0.5, { freq: 300, sweep: 3200, q: 0.7, gain: 0.11 });
            voice(midiHz(52), t0 + 0.04, 0.42, { wave: 'triangle', to: midiHz(76), gain: 0.05 });
        },
        // Врата Hall of Trials, печать: камень дрогнул, металл лопнул.
        // Низкий гул + треск волосяных искр + звон лопнувшей скрепы.
        gateSeal: () => {
            if (!ctx) return;
            const t0 = now();
            noiseHit(t0, 1.5, { freq: 120, sweep: 90, q: 0.5, gain: 0.10 });
            voice(midiHz(31), t0, 1.3, { wave: 'sine', to: midiHz(28), gain: 0.06,
                filter: 'lowpass', cutoff: 400 });
            noiseHit(t0 + 0.62, 0.5, { freq: 5200, sweep: 1400, q: 1.1, gain: 0.05 });
            [86, 79, 74].forEach((m, i) =>
                voice(midiHz(m), t0 + 0.70 + i * 0.05, 0.34, { wave: 'triangle', gain: 0.035 }));
            voice(midiHz(43), t0 + 0.70, 0.7, { wave: 'square', gain: 0.022,
                filter: 'lowpass', cutoff: 900 });
        },
        // Врата Hall of Trials, подъём: скрежет цепи, камень по камню, выдох зала.
        gateOpen: () => {
            if (!ctx) return;
            const t0 = now();
            noiseHit(t0, 1.9, { freq: 420, sweep: 220, q: 0.9, gain: 0.085 });
            for (let k = 0; k < 6; k++) {
                noiseHit(t0 + 0.16 + k * 0.26, 0.22, { freq: 2600, sweep: 800, q: 1.4,
                    gain: 0.045 });
                voice(midiHz(38 + k), t0 + 0.16 + k * 0.26, 0.24, { wave: 'square',
                    gain: 0.02, filter: 'lowpass', cutoff: 900 });
            }
            noiseHit(t0 + 1.05, 1.2, { freq: 260, sweep: 1400, q: 0.6, gain: 0.075 });
            [36, 43, 48, 55].forEach((m, i) =>
                voice(midiHz(m), t0 + 1.35 + i * 0.11, 1.6, { wave: 'triangle',
                    gain: 0.045, attack: 0.12 }));
        },
        // Крафт: металлический аккорд с искрой.
        craft: () => {
            if (!ctx) return;
            const t0 = now();
            [0, 4, 7, 12].forEach((s, i) =>
                voice(midiHz(64 + s), t0 + i * 0.05, 0.3, { wave: 'triangle', gain: 0.05 }));
            noiseHit(t0 + 0.02, 0.2, { freq: 4200, sweep: 1200, gain: 0.05, q: 0.8 });
        },
        // ===== КУЗНЯ (крафт): процесс → успех / провал =====
        // Старт ковки: всасывающий «вууш» и глухой удар по наковальне.
        forgeStart: () => {
            if (!ctx) return;
            const t0 = now();
            noiseHit(t0, 0.42, { freq: 260, sweep: 2600, q: 0.7, gain: 0.075 });
            voice(midiHz(45), t0, 0.34, { wave: 'sawtooth', to: midiHz(64), gain: 0.045,
                filter: 'lowpass', cutoff: 1400 });
        },
        // Тик процесса: молот бьёт по наковальне; тон растёт к финалу (p: 0..1).
        forgeTick: (o) => {
            if (!ctx) return;
            const p = clamp((o && o.p) || 0, 0, 1);
            const t0 = now();
            const base = 58 + p * 15;
            noiseHit(t0, 0.07, { freq: 3600 + p * 2200, sweep: 900, q: 1.2,
                gain: 0.055 + 0.05 * p });
            voice(midiHz(base), t0, 0.13, { wave: 'triangle', gain: 0.04 + 0.02 * p });
            voice(midiHz(base + 12), t0 + 0.004, 0.09, { wave: 'sine', gain: 0.018 });
        },
        // Успех: фанфара вверх; чем выше новая редкость (ri), тем длиннее и ярче.
        craftWin: (o) => {
            if (!ctx) return;
            const ri = clamp((o && o.ri) || 0, 0, 8);
            const t0 = now();
            const oct = Math.min(2, Math.floor(ri / 3)) * 12;
            const notes = [0, 4, 7, 12, 16, 19].slice(0, 4 + Math.min(2, Math.floor(ri / 3)));
            notes.forEach((s, i) => {
                voice(midiHz(64 + s + oct), t0 + i * 0.075, 0.34, {
                    wave: i === notes.length - 1 ? 'triangle' : 'square', gain: 0.05,
                });
            });
            // Искры/звон поверх фанфары — «металл остыл».
            noiseHit(t0 + 0.03, 0.42, { freq: 5400, sweep: 2200, q: 0.7, gain: 0.045 });
            voice(midiHz(88 + oct), t0 + notes.length * 0.075, 0.5, { wave: 'sine', gain: 0.03 });
        },
        // ===== РЕВИЛ ВЫСОКИХ ГРЕЙДОВ (craft_reveal.js) =====
        // Каждый слой сцены звучит своим голосом — по просьбе владельца:
        //  craftRim    — удар по ободу в момент перекраски в цвет петаля;
        //  craftShards — разлёт осколков (звонкая россыпь с панорамой);
        //  craftHum    — гул лучей/вихря (долгий нарос­тающий слой).
        // p — сила ступени 0..1: чем выше грейд, тем ниже и шире удар, тем
        // плотнее гул. Все — мягкие: перегрузка сцены звуком хуже тишины.
        craftRim: (o) => {
            if (!ctx) return;
            const p = clamp((o && o.p) != null ? o.p : 0.5, 0, 1);
            const t0 = now();
            // низкий «бум» — сам обод отзывается массой
            voice(midiHz(33), t0, 0.85 + 0.25 * p, { wave: 'sine', to: midiHz(26),
                gain: 0.085 + 0.05 * p, attack: 0.004 });
            voice(midiHz(45), t0, 0.55, { wave: 'triangle', to: midiHz(38),
                gain: 0.04 + 0.03 * p });
            // металлический звон: три высоких голоса с расстройкой
            [76, 83, 88].forEach((m, i) =>
                voice(midiHz(m + Math.round(p * 5)), t0 + i * 0.012,
                    1.0 - i * 0.14, { wave: 'sine', gain: 0.026 + 0.012 * p,
                        detune: i * 7 }));
            noiseHit(t0, 0.26, { freq: 5200, sweep: 900, q: 1.1, gain: 0.055 + 0.03 * p });
        },
        // Разлёт осколков: короткий «стеклянный» шум и россыпь колких звонов
        // по панораме — осколки летят в разные стороны.
        craftShards: (o) => {
            if (!ctx) return;
            const n = clamp((o && o.n) || 8, 4, 16) | 0;
            const t0 = now();
            noiseHit(t0, 0.32, { freq: 7200, sweep: 2400, q: 0.9, gain: 0.045 });
            for (let i = 0; i < n; i++) {
                const t = t0 + i * 0.026 + Math.random() * 0.02;
                voice(midiHz(84 + Math.round(Math.random() * 12)), t,
                    0.14 + Math.random() * 0.14, { wave: 'sine',
                        gain: 0.016 + Math.random() * 0.012,
                        pan: (Math.random() * 2 - 1) * 0.65 });
            }
        },
        // Гул лучей/вихря: два расстроенных саута с открывающимся фильтром +
        // вибрато-подголосок и шумовой «ветер». Держится пару секунд — это фон
        // под финальной карточкой, а не ещё один удар.
        craftHum: (o) => {
            if (!ctx) return;
            const p = clamp((o && o.p) != null ? o.p : 0.5, 0, 1);
            const dur = 1.6 + 1.2 * p;
            const t0 = now();
            [40, 47].forEach((m, i) =>
                voice(midiHz(m), t0 + i * 0.03, dur, { wave: 'sawtooth',
                    gain: 0.02 + 0.012 * p, attack: 0.32,
                    filter: 'lowpass', cutoff: 420 + 280 * p, filterTo: 900 + 520 * p,
                    detune: i ? 9 : -9 }));
            voice(midiHz(64), t0 + 0.15, dur * 0.68, { wave: 'sine',
                gain: 0.011 + 0.01 * p, attack: 0.5, vib: 5 + 2 * p, vibAmt: 6 });
            noiseHit(t0 + 0.1, dur * 0.8, { freq: 900, sweep: 3400, q: 0.6,
                gain: 0.018 + 0.014 * p });
        },
        // Провал: горн «захлебнулся» — падающий вомп и унылый звон молота.
        craftFail: () => {
            if (!ctx) return;
            const t0 = now();
            noiseHit(t0, 0.3, { freq: 1500, sweep: 170, q: 0.6, gain: 0.085 });
            voice(midiHz(55), t0, 0.34, { wave: 'sawtooth', to: midiHz(38), gain: 0.055,
                filter: 'lowpass', cutoff: 900 });
            [64, 63, 58].forEach((m, i) =>
                voice(midiHz(m), t0 + 0.12 + i * 0.11, 0.26, { wave: 'triangle', gain: 0.038,
                    filter: 'lowpass', cutoff: 1200 }));
        },
        // Поглощение (Absorb): шиммер вверх, высота/яркость — от полученного XP.
        absorb: (o) => {
            if (!ctx) return;
            const p = clamp((o && o.p) != null ? o.p : 0.4, 0, 1);
            const t0 = now();
            noiseHit(t0, 0.5, { freq: 400, sweep: 5200, q: 0.8, gain: 0.06 });
            [0, 7, 12, 16, 19].forEach((s, i) =>
                voice(midiHz(64 + s + Math.round(p * 5)), t0 + i * 0.06, 0.3, {
                    wave: 'sine', gain: 0.042,
                }));
        },
        // Уровень вверх: фанфара — арпеджио вверх с квинтой-подголоском и искрой.
        levelup: () => {
            if (!ctx) return;
            const t0 = now();
            [0, 4, 7, 12, 16].forEach((s, i) => {
                voice(midiHz(60 + s), t0 + i * 0.085, 0.22, { wave: 'square', gain: 0.05 });
                voice(midiHz(53 + s), t0 + i * 0.085, 0.22, { wave: 'triangle',
                    gain: 0.022 });
            });
            noiseHit(t0 + 0.42, 0.35, { freq: 5600, sweep: 2600, q: 0.7, gain: 0.035 });
            voice(midiHz(88), t0 + 0.42, 0.5, { wave: 'sine', gain: 0.03 });
        },
        // Смерть: минорная триада вниз.
        death: () => {
            if (!ctx) return;
            const t0 = now();
            [69, 65, 60, 53].forEach((m, i) =>
                voice(midiHz(m), t0 + i * 0.14, 0.5, { wave: 'triangle', gain: 0.06 }));
        },
        // Респавн: восходящее глиссандо.
        respawn: () => {
            if (!ctx) return;
            const t0 = now();
            voice(midiHz(45), t0, 0.42, { wave: 'sawtooth', to: midiHz(69), gain: 0.05,
                filter: 'lowpass', cutoff: 1800 });
        },
        // Телепорт-варп (формация/рывок).
        warp: () => {
            if (!ctx) return;
            const t0 = now();
            voice(midiHz(72), t0, 0.2, { wave: 'sawtooth', to: midiHz(58), gain: 0.045,
                filter: 'bandpass', cutoff: 1400, q: 1.6 });
        },
    };

    const UI = {
        click: () => { if (ctx) voice(midiHz(86), now(), 0.05, { wave: 'square', gain: 0.045, dest: uiGain }); },
        open: () => {
            if (!ctx) return;
            const t0 = now();
            voice(midiHz(74), t0, 0.07, { wave: 'triangle', gain: 0.05, dest: uiGain });
            voice(midiHz(81), t0 + 0.05, 0.09, { wave: 'triangle', gain: 0.045, dest: uiGain });
        },
        close: () => {
            if (!ctx) return;
            const t0 = now();
            voice(midiHz(81), t0, 0.07, { wave: 'triangle', gain: 0.045, dest: uiGain });
            voice(midiHz(72), t0 + 0.05, 0.09, { wave: 'triangle', gain: 0.04, dest: uiGain });
        },
        error: () => {
            if (!ctx) return;
            const t0 = now();
            voice(midiHz(48), t0, 0.16, { wave: 'square', to: midiHz(46), gain: 0.05, dest: uiGain });
        },
        // Покупка/надевание: «ка-чинг».
        buy: () => {
            if (!ctx) return;
            const t0 = now();
            voice(midiHz(88), t0, 0.08, { wave: 'square', gain: 0.05, dest: uiGain });
            voice(midiHz(95), t0 + 0.07, 0.16, { wave: 'square', gain: 0.05, dest: uiGain });
        },
        equip: () => {
            if (!ctx) return;
            const t0 = now();
            [76, 83, 88].forEach((m, i) =>
                voice(midiHz(m), t0 + i * 0.05, 0.12, { wave: 'triangle', gain: 0.045, dest: uiGain }));
        },
        unequip: () => {
            if (!ctx) return;
            const t0 = now();
            [83, 76].forEach((m, i) =>
                voice(midiHz(m), t0 + i * 0.06, 0.12, { wave: 'triangle', gain: 0.04, dest: uiGain }));
        },
        claim: () => {
            if (!ctx) return;
            const t0 = now();
            [72, 76, 79, 84].forEach((m, i) =>
                voice(midiHz(m), t0 + i * 0.07, 0.18, { wave: 'square', gain: 0.05, dest: uiGain }));
        },
        // Достижение разблокировано: колокольная фанфара — «торжественно», не
        // путать с levelup (тот быстрый и квадратный).
        achv: () => {
            if (!ctx) return;
            const t0 = now();
            [64, 68, 71, 76].forEach((m, i) =>
                voice(midiHz(m), t0 + i * 0.1, 0.4, { wave: 'triangle', gain: 0.045,
                    dest: uiGain }));
            bellAt(88, t0 + 0.4, 1.1, { gain: 0.05, dest: uiGain });
            noiseHit(t0 + 0.4, 0.5, { freq: 5800, sweep: 2400, q: 0.7, gain: 0.03,
                dest: uiGain });
        },
        // Приглашение в отряд: мягкий двухнотный «внимание» (маримба-тембр).
        invite: () => {
            if (!ctx) return;
            const t0 = now();
            voice(midiHz(79), t0, 0.16, { wave: 'triangle', gain: 0.05, dest: uiGain });
            voice(midiHz(84), t0 + 0.16, 0.22, { wave: 'triangle', gain: 0.05, dest: uiGain });
            voice(midiHz(84), t0 + 0.44, 0.18, { wave: 'sine', gain: 0.025, dest: uiGain });
        },
        // Клейм шардов (Tasks): каскад «монеток» — короткие высокие ноты вниз.
        shardClaim: () => {
            if (!ctx) return;
            const t0 = now();
            const scale = [88, 86, 83, 81, 79, 76];
            scale.forEach((m, i) =>
                voice(midiHz(m + (Math.random() < 0.4 ? -1 : 0)), t0 + i * 0.055, 0.12,
                    { wave: 'square', gain: 0.042, dest: uiGain }));
            noiseHit(t0, 0.25, { freq: 6400, sweep: 3200, q: 0.8, gain: 0.025,
                dest: uiGain });
        },
        // Календарь (ежедневная награда): «подарок» — тёплый аккорд + монетки.
        calendarClaim: () => {
            if (!ctx) return;
            const t0 = now();
            noiseHit(t0, 0.4, { freq: 500, sweep: 4200, q: 0.7, gain: 0.035,
                dest: uiGain });
            [0, 4, 7, 12].forEach((s, i) =>
                voice(midiHz(64 + s), t0 + 0.05 + i * 0.02, 0.5, { wave: 'triangle',
                    gain: 0.04, attack: 0.04, dest: uiGain }));
            [88, 92, 95].forEach((m, i) =>
                voice(midiHz(m), t0 + 0.3 + i * 0.07, 0.16, { wave: 'square',
                    gain: 0.04, dest: uiGain }));
        },
    };

    function playSfx(name, opts) {
        if (!enabled) return;
        unlock();
        if (!ctx) return;
        try {
            const fn = SFX[name] || UI[name];
            if (fn) fn(opts || {});
        } catch (e) { /* звук не стоит игры */ }
    }
    function playUi(name, opts) {
        if (!enabled) return;
        unlock();
        if (!ctx) return;
        try {
            const fn = UI[name] || SFX[name];
            if (fn) fn(opts || {});
        } catch (e) { /* ignore */ }
    }

    // ------------------------------------------------------ музыка (секвенсор)
    // v2.0: песня = КОМПОЗИЦИЯ из секций. Секвенсор идёт по тактам секция за
    // секцией, в конце — петля на начало. Каждая секция может переопределить
    // любой слой (паттерны, барабаны, тембр, плотность vel, филл на последнем
    // такте, краш на первом) — так у трека появляется интро, развитие,
    // брейк, кульминация и разрешение вместо бесконечной петли-начала.
    //
    // Поля песни (дефолты для секций): bpm, key, cut, rev, echo, dist,
    // prog (аккорды, полутона от тоники), arp/bass/lead (16 шагов),
    // kick/snare/hat/stab (шаги), pad/padWave/padGain/padIv,
    // arpWave/arpGain/arpOct/arpDur/arpAttack, bassWave/bassGain/bassCut/bassDur,
    // leadWave/leadGain/leadOct/leadDur.
    // Поля секции: bars (обязательно), любые из полей выше + vel (плотность),
    // fill ('light'|'big' — подкат на последнем такте), crash (крэш на старте).

    const _ = null; // пауза в паттерне (для читаемости таблиц)

    // sec — секция композиции: bars тактов + переопределения слоёв.
    function sec(bars, o) { return Object.assign({ bars: bars }, o || {}); }

    const SONGS = {
        // ===== МЕНЮ =====
        // Тёплое и живое: колокольчики-клавиши над струнным пэдом, медленное
        // развитие A → A' с контр-мелодией, спокойное разрешение. Арка 24 такта.
        menu: {
            bpm: 84, key: 57, cut: 3000, rev: 0.34, echo: 0.16,
            prog: [0, -4, -2, 3],
            arpWave: 'bell', arpGain: 0.036, arpOct: 12, arpDur: 0.55,
            bassWave: 'sine', bassGain: 0.058, bassCut: 800, bassDur: 0.32,
            padWave: 'triangle', padGain: 0.02, padIv: [0, 7, 12, 15],
            leadWave: 'sine', leadGain: 0.02, leadOct: 24, leadDur: 0.7,
            sections: [
                // интро: только колокола и пэд
                sec(4, { vel: 0.8,
                    arp: [12, _, _, _, _, _, 7, _, _, _, 4, _, _, _, _, _],
                    bass: [_, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                    kick: [], hat: [], snare: [] }),
                // тема
                sec(8, { vel: 1,
                    arp: [12, _, 7, _, 4, _, _, _, 0, _, 4, _, 7, _, 12, _],
                    bass: [0, _, _, _, _, _, 7, _, _, _, _, _, _, _, _, _],
                    kick: [0, 8], hat: [4, 12], snare: [] }),
                // развитие: вариация арпа + контр-мелодия + лёгкий снейр
                sec(8, { vel: 1.05, fill: 'light',
                    arp: [0, _, 4, _, 7, _, _, _, 12, _, 7, _, 4, _, _, _],
                    lead: [_, _, _, _, 19, _, _, _, _, _, 16, _, _, 14, _, _],
                    bass: [0, _, _, _, 7, _, _, _, _, _, 12, _, _, _, _, _],
                    kick: [0, 8], hat: [4, 6, 12], snare: [12] }),
                // разрешение: тоника, всё стихает → петля в интро
                sec(4, { vel: 0.8, prog: [0],
                    arp: [12, _, _, _, 7, _, _, _, 4, _, _, _, 0, _, _, _],
                    bass: [0, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                    kick: [], hat: [], snare: [] }),
            ],
        },
        // ===== ГАЛЕРЕЯ (музей ночью) =====
        // Редкие колокола в реверб-пустоте. Сохраняем характер, добавляем
        // глубину (rev 0.55) и трёхчастную форму с «нижним» этажом.
        gallery: {
            bpm: 58, key: 50, cut: 2200, rev: 0.55, echo: 0.4,
            prog: [0, -5, 3, -7],
            arpWave: 'bell', arpGain: 0.045, arpOct: 12, arpDur: 1.2, arpAttack: 0.008,
            bassWave: 'sine', bassGain: 0.05, bassCut: 700, bassDur: 1.2,
            padWave: 'sine', padGain: 0.03, padIv: [0, 7, 12, 19],
            sections: [
                sec(6, { vel: 0.9,
                    arp: [12, _, _, _, _, _, _, _, 19, _, _, _, _, _, _, _],
                    bass: [0, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                    kick: [], hat: [], snare: [] }),
                sec(6, { vel: 0.8, prog: [0, -5],
                    arp: [0, _, _, _, _, _, 7, _, _, _, _, _, _, _, _, _],
                    bass: [0, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                    kick: [], hat: [], snare: [] }),
                sec(4, { vel: 0.9,
                    arp: [24, _, _, _, 19, _, _, _, 16, _, _, _, _, _, _, _],
                    bass: [0, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                    kick: [], hat: [], snare: [] }),
            ],
        },
        // ===== МАГАЗИН =====
        // Свой характер (НЕ близнец меню): лаунж с септимами в пэде,
        // офф-бит кик и раскачанный хэт-грув, бас ходит по септимам. Арка 24.
        shop: {
            bpm: 76, key: 53, cut: 2600, rev: 0.3, echo: 0.18,
            prog: [0, 3, 5, -2],
            arpWave: 'triangle', arpGain: 0.03, arpOct: 12, arpDur: 0.2,
            bassWave: 'sine', bassGain: 0.065, bassCut: 700, bassDur: 0.3,
            padWave: 'triangle', padGain: 0.024, padIv: [0, 7, 10, 14],
            sections: [
                sec(4, { vel: 0.8,
                    arp: [7, _, _, _, 10, _, _, _, 14, _, _, _, 10, _, _, _],
                    bass: [_, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                    kick: [], hat: [], snare: [] }),
                sec(8, { vel: 1,
                    arp: [7, _, 10, _, 12, _, 10, _, 7, _, 5, _, _, _, _, _],
                    bass: [0, _, _, _, _, _, 10, _, 7, _, _, _, _, _, _, _],
                    kick: [0, 10], hat: [4, 6, 12, 14], snare: [] }),
                sec(8, { vel: 1.05, fill: 'light',
                    arp: [14, _, 12, _, 10, _, 12, _, 14, _, 17, _, 14, _, 12, _],
                    bass: [0, _, _, 10, _, _, 7, _, 5, _, _, _, 7, _, 10, _],
                    kick: [0, 6, 10], hat: [2, 6, 10, 14], snare: [] }),
                sec(4, { vel: 0.75, prog: [0],
                    arp: [12, _, _, _, 10, _, _, _, 7, _, _, _, 5, _, _, _],
                    bass: [0, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                    kick: [], hat: [], snare: [] }),
            ],
        },
        // ===== РЕДАКТОР АВАТАРА (WARDROBE) =====
        // Почти эмбиент: дрон-пэд и редкие ноты, вторая половина дышит глубже.
        avatar: {
            bpm: 52, key: 60, cut: 1700, rev: 0.5, echo: 0.3,
            prog: [0, 0, -3, -5],
            arpWave: 'sine', arpGain: 0.032, arpOct: 12, arpDur: 1.4, arpAttack: 0.05,
            bassWave: 'sine', bassGain: 0.045, bassCut: 600, bassDur: 1.2,
            padWave: 'triangle', padGain: 0.028, padIv: [0, 7, 12, 15],
            sections: [
                sec(8, { vel: 0.85,
                    arp: [7, _, _, _, _, _, _, _, 12, _, _, _, _, _, _, _],
                    bass: [_, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                    kick: [], hat: [], snare: [] }),
                sec(8, { vel: 0.95,
                    arp: [7, _, _, _, _, _, 12, _, _, _, _, _, 15, _, _, _],
                    bass: [0, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                    kick: [], hat: [], snare: [] }),
            ],
        },
        // ===== ЭКРАН РЕАЛМА =====
        // Торжественная: колокольное интро → маршевая тема → гранд с контр-
        // мелодией и 16-ми → колокольное разрешение.
        realm: {
            bpm: 92, key: 52, cut: 2900, rev: 0.35, echo: 0.2,
            prog: [0, 3, 5, -2],
            arpWave: 'square', arpGain: 0.022, arpOct: 12, arpDur: 0.13,
            bassWave: 'sawtooth', bassGain: 0.06, bassCut: 800, bassDur: 0.2,
            padWave: 'triangle', padGain: 0.02, padIv: [0, 7, 12, 15],
            leadWave: 'sine', leadGain: 0.02, leadOct: 24, leadDur: 0.6,
            sections: [
                sec(4, { vel: 0.85, arpWave: 'bell', arpGain: 0.04, arpDur: 0.5,
                    arp: [12, _, _, _, 19, _, _, _, 16, _, _, _, _, _, _, _],
                    bass: [_, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                    kick: [], hat: [], snare: [] }),
                sec(8, { vel: 1,
                    arp: [0, 7, 12, 7, 3, 10, 15, 10, 5, 12, 17, 12, -2, 5, 10, 5],
                    bass: [0, 0, _, 7, _, 3, 3, _, 5, 5, _, 12, -2, -2, _, 5],
                    kick: [0, 6, 8, 14], hat: [2, 10], snare: [4, 12] }),
                sec(8, { vel: 1.08, arpOct: 24, fill: 'light',
                    arp: [0, 7, 12, 7, 3, 10, 15, 10, 5, 12, 17, 12, -2, 5, 10, 5],
                    lead: [_, _, 12, _, _, _, _, _, _, _, 7, _, _, _, _, _],
                    bass: [0, 0, _, 7, _, 3, 3, _, 5, 5, _, 12, -2, -2, _, 5],
                    kick: [0, 4, 8, 12], hat: [2, 6, 10, 14], snare: [4, 12] }),
                sec(4, { vel: 0.85, prog: [0], arpWave: 'bell', arpGain: 0.04,
                    arpOct: 12, arpDur: 0.5,
                    arp: [12, _, _, 19, _, _, 16, _, _, _, _, _, _, _, _, _],
                    bass: [0, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                    kick: [], hat: [], snare: [] }),
            ],
        },
        // ===== МИР (запасной трек) =====
        // Драйв для кастомных зон: интро-пульс → тема → брейк → тема с лидом.
        world: {
            bpm: 128, key: 50, cut: 3400, rev: 0.12, echo: 0.08, dist: 0.3,
            prog: [0, 0, 5, 3],
            arpWave: 'sawtooth', arpGain: 0.02, arpOct: 12, arpDur: 0.13,
            bassWave: 'sawtooth', bassGain: 0.075, bassCut: 620, bassDur: 0.16,
            padWave: 'triangle', padGain: 0.016, padIv: [0, 7, 12, 15],
            leadWave: 'square', leadGain: 0.016, leadOct: 24, leadDur: 0.2,
            sections: [
                sec(4, { vel: 0.85,
                    arp: [0, _, _, _, 7, _, _, _, 12, _, _, _, 7, _, _, _],
                    bass: [0, _, _, _, _, _, _, _, 0, _, _, _, _, _, _, _],
                    kick: [0, 8], hat: [4, 12], snare: [] }),
                sec(8, { vel: 1, crash: true,
                    arp: [0, 4, 7, 4, 12, 7, 4, 7, 0, 4, 7, 4, 14, 7, 4, 7],
                    bass: [0, _, 0, _, 7, _, _, _, 5, _, 5, _, 3, _, 3, _],
                    kick: [0, 4, 8, 12], hat: [2, 6, 10, 14], snare: [4, 12] }),
                sec(4, { vel: 0.7,
                    arp: [0, _, _, _, _, _, 12, _, _, _, _, _, 7, _, _, _],
                    bass: [0, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                    kick: [], hat: [4, 12], snare: [], pad: true }),
                sec(8, { vel: 1.05, fill: 'light',
                    arp: [0, 4, 7, 4, 12, 7, 4, 7, 0, 4, 7, 4, 14, 7, 4, 7],
                    lead: [_, _, _, _, 12, _, _, _, _, _, _, _, 14, _, 12, _],
                    bass: [0, _, 0, _, 7, _, _, _, 5, _, 5, _, 3, _, 3, _],
                    kick: [0, 4, 8, 12], hat: [2, 6, 10, 14], snare: [4, 12] }),
            ],
        },

        // ===== HALL OF TRIALS: древний эмбиент зала =====
        // Отдельная тема, не ступень Origin: игрок ВНУТРИ холла (myHallTier > 0)
        // слышит медленный суб-дрон, далёкий колокол, хор-пэд на квинтах и
        // редкое сердце-тамтам. Без марша и без «святой» мажорности ступени:
        // это склеп, в котором огонь — единственный источник света.
        hall: {
            bpm: 46, key: 43, cut: 1250, rev: 0.66, echo: 0.34, delay: 0.46,
            prog: [0, -2, 3, -5],
            arpWave: 'bell', arpGain: 0.05, arpOct: 24, arpDur: 2.8, arpAttack: 0.005,
            bassWave: 'sine', bassGain: 0.09, bassCut: 260, bassDur: 3.6,
            padWave: 'sine', padGain: 0.042, padIv: [0, 7, 12, 19],
            leadWave: 'triangle', leadGain: 0.012, leadOct: 12, leadDur: 3.2,
            sections: [
                // 8 тактов: только дрон и одиночный колокол в глубине.
                sec(8, { vel: 0.85, pad: true,
                    arp: [_, _, _, _, _, _, _, _, 12, _, _, _, _, _, _, _],
                    bass: [0, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                    kick: [], hat: [], snare: [] }),
                // 8 тактов: низкий там-там как сердце, редкие колокола.
                sec(8, { vel: 0.95, pad: true,
                    arp: [_, _, _, _, 7, _, _, _, _, _, _, _, 3, _, _, _],
                    bass: [0, _, _, _, _, _, _, _, -5, _, _, _, _, _, _, _],
                    kick: [0, 10], hat: [], snare: [] }),
                // 4 такта: хор поднимается, литавра в конце.
                sec(4, { vel: 1.05, pad: true, fill: 'big', crash: true,
                    arp: [12, _, _, _, 19, _, _, _, _, _, _, _, 15, _, _, _],
                    bass: [0, _, _, _, 5, _, _, _, _, _, _, _, _, _, _, _],
                    kick: [0, 8], hat: [], snare: [] }),
                // 8 тактов: полная тема зала — колокол, контр-голос, дрон.
                sec(8, { vel: 1.0, pad: true,
                    arp: [19, _, _, _, 12, _, _, _, 15, _, _, _, 12, _, _, _],
                    lead: [_, _, _, _, _, _, 12, _, _, _, _, _, _, _, 7, _],
                    bass: [0, _, _, _, -2, _, _, _, -5, _, _, _, 3, _, _, _],
                    kick: [0, 6, 10], hat: [], snare: [] }),
            ],
        },
    };

    // ===== СТУПЕНИ БИОМОВ (TIERS) =====
    // Мир идёт по ступеням: чем дальше, тем плотнее и злее музыка. Каждая
    // ступень — композиция со своей аркой; эскалация плотности сохранена.
    //   0 Spawn  — колыбельная   1 Origin — «святое»   2 Point — тёплая/живая
    //   3 Line   — пульс         4 Triangle — уверенно 5 Square — драйв
    //   6 Pentagon — овердрайв   7 Hexagon — синкопы   8 Fractal — жёстко
    //   9 Chaos  — ЭПИЧЕСКАЯ АРКА (интро→билд→рифф→брейк→кульминация→конклюзия)
    //  10 Nexus  — хаб: торжественно
    const TIERS = [
        // 0 — Spawn: колыбельная (20 тактов)
        { bpm: 56, key: 48, cut: 1500, rev: 0.5, echo: 0.26,
          prog: [0, 0, -3, -5],
          arpWave: 'sine', arpGain: 0.034, arpOct: 12, arpDur: 0.5, arpAttack: 0.05,
          bassWave: 'sine', bassGain: 0.05, bassCut: 600, bassDur: 1.1,
          padWave: 'sine', padGain: 0.034, padIv: [0, 7, 12, 15],
          sections: [
            sec(8, { vel: 0.9,
                arp: [0, _, _, _, _, _, 7, _, _, _, _, _, 5, _, _, _],
                bass: [_, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                kick: [], hat: [], snare: [] }),
            sec(8, { vel: 1,
                arp: [0, _, _, _, _, _, 7, _, _, _, 12, _, _, _, _, _],
                bass: [0, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                kick: [], hat: [], snare: [] }),
            sec(4, { vel: 0.8,
                arp: [0, _, _, _, _, _, 7, _, _, _, _, _, 5, _, _, _],
                bass: [_, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                kick: [], hat: [], snare: [] }),
          ] },
        // 1 — Origin: колокольный, «святой» (20 тактов)
        { bpm: 60, key: 55, cut: 1900, rev: 0.55, echo: 0.3,
          prog: [0, 5, 7, 3],
          arpWave: 'bell', arpGain: 0.04, arpOct: 12, arpDur: 0.44, arpAttack: 0.006,
          bassWave: 'sine', bassGain: 0.05, bassCut: 650, bassDur: 1.1,
          padWave: 'sine', padGain: 0.03, padIv: [0, 7, 12, 19],
          sections: [
            sec(8, { vel: 0.9,
                arp: [12, _, _, _, 19, _, _, _, _, _, _, _, 16, _, _, _],
                bass: [0, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                kick: [], hat: [], snare: [] }),
            sec(8, { vel: 1,
                arp: [19, _, _, _, 24, _, _, _, _, _, 23, _, _, _, _, _],
                bass: [0, _, _, _, _, _, _, _, 7, _, _, _, _, _, _, _],
                kick: [], hat: [], snare: [] }),
            sec(4, { vel: 0.8,
                arp: [12, _, _, _, _, _, _, _, 19, _, _, _, _, _, _, _],
                bass: [0, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                kick: [], hat: [], snare: [] }),
          ] },
        // 2 — Point: тёплая, живая (24 такта) — «визитка» первых зон
        { bpm: 66, key: 55, cut: 2300, rev: 0.4, echo: 0.22,
          prog: [0, -4, -2, 3],
          arpWave: 'bell', arpGain: 0.038, arpOct: 12, arpDur: 0.42,
          bassWave: 'triangle', bassGain: 0.055, bassCut: 750, bassDur: 0.3,
          padWave: 'triangle', padGain: 0.028, padIv: [0, 7, 12, 15],
          sections: [
            sec(4, { vel: 0.85,
                arp: [7, _, _, _, 12, _, _, _, _, _, _, _, _, _, _, _],
                bass: [_, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                kick: [], hat: [], snare: [] }),
            sec(8, { vel: 1, crash: true,
                arp: [7, _, 12, _, _, 10, _, 12, 7, _, 12, _, 15, _, 12, _],
                bass: [0, _, _, _, 7, _, _, _, 3, _, _, _, -2, _, _, 5],
                kick: [], hat: [10], snare: [] }),
            sec(8, { vel: 1.02, fill: 'light',
                arp: [7, _, 12, _, _, 15, _, 12, 7, _, 10, _, 12, _, _, _],
                bass: [0, _, _, 7, _, _, _, _, -2, _, _, 5, _, _, 3, _],
                kick: [0, 8], hat: [4, 10, 12], snare: [] }),
            sec(4, { vel: 0.8, prog: [0],
                arp: [7, _, _, _, 12, _, _, _, _, _, _, _, _, _, _, _],
                bass: [0, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                kick: [], hat: [], snare: [] }),
          ] },
        // 3 — Line: спокойно, но с пульсом (24 такта)
        { bpm: 72, key: 57, cut: 2400, rev: 0.35, echo: 0.2,
          prog: [0, -2, 3, -4],
          arpWave: 'triangle', arpGain: 0.028, arpOct: 12, arpDur: 0.16,
          bassWave: 'triangle', bassGain: 0.06, bassCut: 750, bassDur: 0.24,
          padWave: 'triangle', padGain: 0.026, padIv: [0, 7, 12, 15],
          sections: [
            sec(4, { vel: 0.85,
                arp: [0, _, _, _, 4, _, _, _, 7, _, _, _, _, _, _, _],
                bass: [_, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                kick: [], hat: [], snare: [] }),
            sec(8, { vel: 1, crash: true,
                arp: [0, _, 4, _, 7, _, 4, 12, 0, _, 4, _, 9, _, 7, 4],
                bass: [0, _, _, _, 7, _, _, _, 3, _, _, _, -2, _, _, 5],
                kick: [0, 8], hat: [4, 12], snare: [] }),
            sec(8, { vel: 1.04, fill: 'light',
                arp: [12, _, 7, _, 4, _, 7, 12, 9, _, 7, _, 4, _, _, _],
                bass: [0, _, _, 7, _, _, _, _, -2, _, _, 5, _, _, 7, _],
                kick: [0, 8], hat: [4, 6, 12], snare: [12] }),
            sec(4, { vel: 0.8, prog: [0],
                arp: [0, _, _, _, 4, _, _, _, 7, _, _, _, _, _, _, _],
                bass: [0, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                kick: [], hat: [], snare: [] }),
          ] },
        // 4 — Triangle: уверенный средний темп (24 такта)
        { bpm: 78, key: 52, cut: 2600, rev: 0.3, echo: 0.18,
          prog: [0, 3, 5, -2],
          arpWave: 'square', arpGain: 0.026, arpOct: 12, arpDur: 0.14,
          bassWave: 'triangle', bassGain: 0.062, bassCut: 700, bassDur: 0.2,
          padWave: 'triangle', padGain: 0.024, padIv: [0, 7, 12, 15],
          sections: [
            sec(4, { vel: 0.85, arpWave: 'triangle',
                arp: [0, _, _, _, 7, _, _, _, 12, _, _, _, _, _, _, _],
                bass: [_, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                kick: [0, 8], hat: [4, 12], snare: [] }),
            sec(8, { vel: 1, crash: true,
                arp: [0, 7, 4, 7, 12, 7, 4, 7, 3, 10, 7, 10, 15, 10, 7, 10],
                bass: [0, _, 0, _, 7, _, 7, _, 3, _, 3, _, 5, _, 5, _],
                kick: [0, 8], hat: [2, 6, 10, 14], snare: [] }),
            sec(8, { vel: 1.04, fill: 'light',
                arp: [12, 7, 4, 7, 3, 10, 7, 10, 15, 10, 7, 10, 12, 14, 15, _],
                bass: [0, _, 0, _, 7, _, 7, _, 3, _, 3, _, 5, _, 5, 7],
                kick: [0, 6, 8], hat: [2, 6, 10, 14], snare: [4, 12] }),
            sec(4, { vel: 0.8, prog: [0], arpWave: 'triangle',
                arp: [12, _, _, _, 7, _, _, _, _, _, _, _, _, _, _, _],
                bass: [0, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                kick: [0, 8], hat: [4, 12], snare: [] }),
          ] },
        // 5 — Square: драйв + бэкбит (24 такта)
        { bpm: 86, key: 50, cut: 2800, rev: 0.25, echo: 0.16,
          prog: [0, 0, 5, 3],
          arpWave: 'square', arpGain: 0.024, arpOct: 12, arpDur: 0.13,
          bassWave: 'sawtooth', bassGain: 0.066, bassCut: 680, bassDur: 0.18,
          padWave: 'triangle', padGain: 0.02, padIv: [0, 7, 12, 15],
          sections: [
            sec(4, { vel: 0.85, arpWave: 'triangle', arpGain: 0.03,
                arp: [0, _, _, _, 7, _, _, _, 12, _, _, _, _, _, _, _],
                bass: [_, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                kick: [0, 8], hat: [4, 12], snare: [] }),
            sec(8, { vel: 1, crash: true,
                arp: [0, 7, 12, 7, 4, 12, 16, 12, 0, 7, 12, 7, 5, 12, 14, 12],
                bass: [0, _, 0, _, 7, _, 7, _, 5, _, 5, _, 3, _, 3, 7],
                kick: [0, 6, 8, 14], hat: [2, 6, 10, 14], snare: [4, 12] }),
            sec(8, { vel: 1.05, fill: 'light',
                arp: [16, 12, 7, 12, 14, 12, 7, 12, 0, 7, 12, 7, 5, 12, 14, 12],
                bass: [0, _, 0, 0, 7, _, 7, _, 5, _, 5, 5, 3, _, 3, 7],
                kick: [0, 6, 8, 14], hat: [2, 6, 10, 14], snare: [4, 12] }),
            sec(4, { vel: 0.8, prog: [0], arpWave: 'triangle', arpGain: 0.03,
                arp: [12, _, _, _, 7, _, _, _, _, _, _, _, _, _, _, _],
                bass: [0, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                kick: [0, 8], hat: [4, 12], snare: [] }),
          ] },
        // 6 — Pentagon: плотнее, лёгкий овердрайв (24 такта)
        { bpm: 94, key: 50, cut: 3000, rev: 0.2, echo: 0.12, dist: 0.12,
          prog: [0, -2, 5, 3],
          arpWave: 'sawtooth', arpGain: 0.022, arpOct: 12, arpDur: 0.12,
          bassWave: 'sawtooth', bassGain: 0.07, bassCut: 660, bassDur: 0.16,
          padWave: 'triangle', padGain: 0.016, padIv: [0, 7, 12, 15],
          sections: [
            sec(4, { vel: 0.85, arpWave: 'square',
                arp: [0, _, _, _, 7, _, _, _, 12, _, _, _, _, _, _, _],
                bass: [_, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                kick: [0, 8], hat: [4, 12], snare: [] }),
            sec(8, { vel: 1, crash: true,
                arp: [0, 7, 12, 14, 12, 7, 12, 16, 0, 7, 12, 14, 19, 14, 12, 7],
                bass: [0, 0, _, 7, _, 7, _, 0, 5, 5, _, 10, _, 5, _, 3],
                kick: [0, 4, 8, 14], hat: [2, 6, 10, 14], snare: [4, 12] }),
            sec(8, { vel: 1.05, fill: 'light',
                arp: [19, 14, 12, 14, 16, 14, 12, 14, 0, 7, 12, 14, 19, 14, 12, 7],
                bass: [0, 0, _, 7, _, 7, _, 0, 5, 5, _, 10, _, 5, 5, 3],
                kick: [0, 4, 8, 12, 14], hat: [2, 6, 10, 14], snare: [4, 12] }),
            sec(4, { vel: 0.8, prog: [0], arpWave: 'square',
                arp: [12, _, _, _, 7, _, _, _, _, _, _, _, _, _, _, _],
                bass: [0, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                kick: [0, 8], hat: [4, 12], snare: [] }),
          ] },
        // 7 — Hexagon: синкопы, шестнадцатые (24 такта)
        { bpm: 102, key: 45, cut: 3200, rev: 0.15, echo: 0.1, dist: 0.24,
          prog: [0, 3, -2, 5],
          arpWave: 'sawtooth', arpGain: 0.02, arpOct: 12, arpDur: 0.11,
          bassWave: 'sawtooth', bassGain: 0.072, bassCut: 650, bassDur: 0.14,
          padWave: 'sawtooth', padGain: 0.012, padIv: [0, 7, 12, 15],
          sections: [
            sec(4, { vel: 0.85, arpWave: 'square',
                arp: [0, _, _, _, 7, _, _, _, 12, _, _, _, _, _, _, _],
                bass: [_, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                kick: [0, 8], hat: [4, 12], snare: [] }),
            sec(8, { vel: 1, crash: true,
                arp: [0, 12, 7, 12, 15, 12, 7, 12, 0, 12, 7, 12, 14, 12, 7, 19],
                bass: [0, _, 0, 7, _, 0, 7, _, 5, _, 5, 10, _, 5, 10, _],
                kick: [0, 3, 8, 11], hat: [1, 3, 5, 7, 9, 11, 13, 15],
                snare: [4, 12] }),
            sec(8, { vel: 1.06, fill: 'big',
                arp: [15, 12, 7, 12, 19, 12, 7, 12, 0, 12, 7, 12, 14, 12, 7, 19],
                bass: [0, _, 0, 7, _, 0, 7, 0, 5, _, 5, 10, _, 5, 10, 10],
                kick: [0, 3, 8, 11], hat: [1, 3, 5, 7, 9, 11, 13, 15],
                snare: [4, 12] }),
            sec(4, { vel: 0.8, prog: [0], arpWave: 'square',
                arp: [12, _, _, _, 7, _, _, _, _, _, _, _, _, _, _, _],
                bass: [0, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                kick: [0, 8], hat: [4, 12], snare: [] }),
          ] },
        // 8 — Fractal: жёстко (28 тактов)
        { bpm: 110, key: 44, cut: 3600, rev: 0.1, echo: 0.07, dist: 0.4,
          prog: [0, 5, -2, 3],
          arpWave: 'sawtooth', arpGain: 0.018, arpOct: 12, arpDur: 0.1,
          bassWave: 'sawtooth', bassGain: 0.075, bassCut: 640, bassDur: 0.13,
          padWave: 'sawtooth', padGain: 0.01, padIv: [0, 7, 12, 15],
          sections: [
            sec(4, { vel: 0.85, arpWave: 'square',
                arp: [0, _, _, _, 7, _, _, _, 12, _, _, _, _, _, _, _],
                bass: [_, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                kick: [0, 8], hat: [4, 12], snare: [] }),
            sec(8, { vel: 1, crash: true,
                arp: [0, 12, 7, 12, 3, 12, 7, 12, 15, 12, 7, 12, 5, 12, 7, 12],
                bass: [0, 0, 7, 0, 0, 0, 7, 10, 5, 5, 12, 5, 5, 5, 12, 3],
                kick: [0, 2, 8, 10, 14], hat: [2, 6, 10, 14], snare: [4, 12] }),
            sec(8, { vel: 1.05, fill: 'big',
                arp: [15, 12, 7, 12, 19, 12, 7, 12, 15, 12, 7, 12, 5, 12, 7, 12],
                bass: [0, 0, 7, 0, 0, 0, 7, 10, 5, 5, 12, 5, 5, 5, 12, 3],
                kick: [0, 2, 8, 10, 14], hat: [1, 3, 5, 7, 9, 11, 13, 15],
                snare: [4, 12] }),
            sec(4, { vel: 0.7, prog: [0],
                arp: [0, _, _, _, _, _, 7, _, _, _, _, _, 12, _, _, _],
                bass: [0, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                kick: [], hat: [4, 12], snare: [], pad: true }),
            sec(8, { vel: 1.1, fill: 'big',
                arp: [0, 12, 7, 12, 3, 12, 7, 12, 15, 12, 7, 12, 5, 12, 7, 12],
                bass: [0, 0, 7, 0, 0, 0, 7, 10, 5, 5, 12, 5, 5, 5, 12, 3],
                kick: [0, 2, 4, 8, 10, 12, 14], hat: [1, 3, 5, 7, 9, 11, 13, 15],
                snare: [4, 12] }),
            sec(4, { vel: 0.8, prog: [0],
                arp: [12, _, _, _, 7, _, _, _, _, _, _, _, _, _, _, _],
                bass: [0, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                kick: [0, 8], hat: [4, 12], snare: [] }),
          ] },
        // 9 — Chaos: ЭПИЧЕСКАЯ АРКА (40 тактов ≈ 76 секунд):
        // интро (чистый тон) → билд (вход барабанов) → главный рифф (дистошн,
        // пауэр-аккорды) → рифф с лидом → брейкдаун → кульминация (дабл-тайм)
        // → конклюзия (разрешение в тонику) → петля.
        { bpm: 126, key: 40, cut: 4300, rev: 0.08, echo: 0.05, dist: 0.55,
          prog: [0, 0, -5, -2],
          arpWave: 'sawtooth', arpGain: 0.016, arpOct: 12, arpDur: 0.11,
          bassWave: 'sawtooth', bassGain: 0.08, bassCut: 640, bassDur: 0.14,
          padWave: 'sawtooth', padGain: 0.012, padIv: [0, 7, 12, 15],
          leadWave: 'sawtooth', leadGain: 0.016, leadOct: 24, leadDur: 0.3,
          stabGain: 1,
          sections: [
            // 1) ИНТРО: чистый тон, только пэд и одиночный арп — «затишье».
            sec(4, { vel: 0.6, prog: [0], arpWave: 'triangle', arpGain: 0.024,
                arp: [0, _, _, 7, _, _, _, _, 12, _, _, _, _, _, _, _],
                bass: [_, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                kick: [], hat: [], snare: [], pad: true, stab: [] }),
            // 2) БИЛД: входит чаггинг-бас, кик на доли, хэты восьмыми; в конце — рол.
            sec(4, { vel: 0.8,
                arp: [_, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                bass: [0, _, 0, _, 0, _, 0, _, 0, _, 0, _, 0, _, 0, _],
                kick: [0, 8], hat: [4, 6, 10, 12, 14], snare: [],
                stab: [], fill: 'big' }),
            // 3) ГЛАВНЫЙ РИФФ: полная плотность — стабы, чаггинг, боевые барабаны.
            sec(8, { vel: 1, crash: true,
                arp: [0, 0, 12, 0, 0, 12, 0, 7, 0, 0, 12, 0, 15, 12, 0, 7],
                bass: [0, 0, 0, 0, 7, 7, 0, 0, 5, 5, 5, 5, 3, 3, 7, 7],
                kick: [0, 4, 8, 10, 12], snare: [4, 12], hat: [2, 6, 10, 14],
                stab: [0, 6, 8, 14] }),
            // 4) РИФФ + ЛИД: та же тема с контр-мелодией сверху.
            sec(8, { vel: 1.05,
                arp: [0, 0, 12, 0, 0, 12, 0, 7, 0, 0, 12, 0, 15, 12, 0, 7],
                lead: [_, _, _, _, 12, _, 10, _, _, _, 7, _, _, _, 10, _],
                bass: [0, 0, 0, 0, 7, 7, 0, 0, 5, 5, 5, 5, 3, 3, 7, 7],
                kick: [0, 4, 8, 10, 12], snare: [4, 12], hat: [2, 6, 10, 14],
                stab: [0, 6, 8, 14] }),
            // 5) БРЕЙКДАУН: барабаны падают, бас держит ноту, пэд возвращается.
            sec(4, { vel: 0.65, prog: [0, 0, -5, -2],
                arp: [12, _, _, _, _, _, _, _, 15, _, _, _, _, _, _, _],
                bass: [0, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                bassDur: 1.2, kick: [0], hat: [], snare: [], stab: [],
                pad: true, fill: 'big' }),
            // 6) КУЛЬМИНАЦИЯ: дабл-тайм — кик четвертями, хэты 16-ми, лид выше.
            sec(8, { vel: 1.12, crash: true,
                arp: [0, 0, 12, 0, 0, 12, 0, 7, 0, 0, 12, 0, 15, 12, 0, 7],
                lead: [19, _, _, _, _, _, 19, _, _, _, 17, _, _, _, _, _],
                bass: [0, 0, 0, 0, 7, 7, 0, 0, 5, 5, 5, 5, 3, 3, 7, 7],
                kick: [0, 4, 8, 12, 14], snare: [4, 12],
                hat: [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15],
                stab: [0, 4, 8, 12], fill: 'big' }),
            // 7) КОНКЛЮЗИЯ: тоника, стаб раз в такт, всё стихает — разрешение.
            sec(4, { vel: 0.8, prog: [0],
                arp: [0, _, _, _, _, _, 7, _, _, _, 12, _, _, _, _, _],
                arpWave: 'triangle', arpGain: 0.022,
                bass: [0, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                bassDur: 0.5, kick: [0, 8], hat: [4, 12], snare: [],
                stab: [0], pad: true, crash: true }),
          ] },
        // 10 — Nexus: хаб, торжественно (20 тактов)
        { bpm: 74, key: 57, cut: 2500, rev: 0.45, echo: 0.26,
          prog: [0, 5, 3, -2],
          arpWave: 'bell', arpGain: 0.036, arpOct: 12, arpDur: 0.4, arpAttack: 0.006,
          bassWave: 'triangle', bassGain: 0.058, bassCut: 700, bassDur: 0.3,
          padWave: 'triangle', padGain: 0.028, padIv: [0, 7, 12, 19],
          sections: [
            sec(4, { vel: 0.85,
                arp: [12, _, _, _, _, _, _, _, 19, _, _, _, _, _, _, _],
                bass: [_, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                kick: [], hat: [], snare: [] }),
            sec(8, { vel: 1, crash: true,
                arp: [12, _, _, 19, _, 16, _, _, 12, _, _, 19, _, 24, _, 16],
                bass: [0, _, _, 7, _, _, 5, _, 5, _, _, 12, _, _, 7, _],
                kick: [0, 8], hat: [12], snare: [] }),
            sec(4, { vel: 1.02, fill: 'light',
                arp: [24, _, _, _, 19, _, 16, _, 12, _, _, 19, _, 24, _, 28],
                bass: [0, _, _, 7, _, _, 5, _, 5, _, _, 12, _, _, 7, _],
                kick: [0, 6, 8], hat: [4, 12], snare: [12] }),
            sec(4, { vel: 0.8, prog: [0],
                arp: [12, _, _, _, 19, _, _, _, _, _, _, _, _, _, _, _],
                bass: [0, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _],
                kick: [], hat: [], snare: [] }),
          ] },
    ];

    // Семья зоны → ступень. Имена зон приходят с сервера (world.js zoneNames),
    // кастомные зоны из редактора могут не совпасть — тогда играет world.
    const FAMILY_TIER = {
        spawn: 0, testroom: 0, tutorial: 0,
        origin: 1,
        point: 2, line: 3, triangle: 4, square: 5,
        pentagon: 6, hexagon: 7, fractal: 8, chaos: 9,
        nexus: 10,
    };
    const ROMAN = { I: 1, II: 2, III: 3, IV: 4, V: 5 };

    // «Chaos II» → { base: 'chaos', variant: 1 }, «Origin» → variant 0.
    function familyOf(name) {
        const s = String(name || '').trim();
        if (!s) return null;
        const m = s.match(/^(.+?)(?:\s+(I|II|III|IV|V))?$/i);
        const base = (m ? m[1] : s).trim().toLowerCase();
        if (FAMILY_TIER[base] == null) return null;
        const roman = (m && m[2]) ? (ROMAN[m[2].toUpperCase()] || 1) : 1;
        return { base: base, variant: Math.max(0, roman - 1) };
    }

    // Вариант «II»/«III» — та же композиция, но узнаваемо другая: транспонирование
    // ключа, лёгкий сдвиг темпа/среза и поворот аккордовой прогрессии в каждой
    // секции (порядок секций и разрешение в тонику сохраняются — арка не ломается).
    function mkZoneSong(family, variant) {
        const t = TIERS[FAMILY_TIER[family]];
        const v = clamp(variant | 0, 0, 4);
        const shift = [0, 5, -2, 7][v % 4];
        if (v === 0) return t;
        const rot = (arr, n) => arr.slice(n).concat(arr.slice(0, n));
        return {
            bpm: Math.round(t.bpm * (1 + 0.03 * v)),
            key: t.key + shift,
            cut: Math.round((t.cut || 2400) * (1 + 0.06 * v)),
            rev: t.rev, echo: t.echo, dist: t.dist || 0,
            // слои-дефолты копируем как есть (паттерны относительны корню,
            // транспонируется сам key)
            prog: t.prog, arp: t.arp, arpWave: t.arpWave, arpGain: t.arpGain,
            arpOct: t.arpOct, arpDur: t.arpDur, arpAttack: t.arpAttack,
            bass: t.bass, bassWave: t.bassWave, bassGain: t.bassGain,
            bassCut: t.bassCut, bassDur: t.bassDur,
            kick: t.kick, snare: t.snare, hat: t.hat, stab: t.stab,
            pad: t.pad, padGain: t.padGain, padWave: t.padWave, padIv: t.padIv,
            lead: t.lead, leadWave: t.leadWave, leadGain: t.leadGain,
            leadOct: t.leadOct, leadDur: t.leadDur, stabGain: t.stabGain,
            sections: t.sections.map((s, si) => ({
                bars: s.bars,
                vel: s.vel, fill: s.fill, crash: s.crash,
                pad: s.pad, padDur: s.padDur, bassDur: s.bassDur,
                arpWave: s.arpWave, arpGain: s.arpGain, arpOct: s.arpOct,
                arpDur: s.arpDur, arpAttack: s.arpAttack,
                lead: s.lead, leadOct: s.leadOct,
                kick: s.kick, hat: s.hat, snare: s.snare, stab: s.stab,
                bass: s.bass, arp: s.arp,
                // прогрессия секции поворачивается — гармония другая, арка та же
                prog: s.prog ? rot(s.prog, (v + si) % s.prog.length) : undefined,
            })),
        };
    }

    // Ключ трека для имени зоны (с кэшем: на портале песня не пересобирается).
    function zoneSongKey(name) {
        const f = familyOf(name);
        if (!f) return null;
        const key = 'z:' + f.base + ':' + f.variant;
        if (!SONGS[key]) SONGS[key] = mkZoneSong(f.base, f.variant);
        return key;
    }

    // Индекс моей зоны и её имя (обе переменные объявлены в других файлах
    // клиента; читаем аккуратно — аудио не имеет права ронять игру).
    function zoneIndex() {
        try { return (typeof myZone === 'number') ? myZone : -1; } catch (e) { return -1; }
    }
    function zoneNameAt(idx) {
        try {
            if (typeof zones !== 'undefined' && zones && zones[idx]) return zones[idx].name || '';
        } catch (e) { /* ignore */ }
        return '';
    }
    // Настроение текущей зоны: в холле испытаний — своя древняя тема; иначе
    // трек ступени биома (или world для кастомных зон).
    function moodForZone(idx) {
        try {
            if (typeof myHallTier === 'number' && myHallTier > 0) return 'hall';
        } catch (e) { /* ignore */ }
        return zoneSongKey(zoneNameAt(idx)) || 'world';
    }

    // Музыкальные экраны: пока такой открыт, фон мира молчит (лавка поверх мира
    // играет свою «торговую» тему, а не трек зоны).
    const MUSIC_SCREENS = ['menu', 'realmScreen', 'galleryScreen', 'shopScreen', 'avatarScreen'];
    function screenMusicBusy() {
        for (const id of MUSIC_SCREENS) {
            const el = document.getElementById(id);
            if (el && el.classList.contains('active')) return true;
        }
        return false;
    }

    // followZone — прошли портал → другая ступень, другая музыка. Смотрим
    // только когда игрок В МИРЕ (myZone >= 0). Вторая проверка — страховка от
    // чужих showScreen('menu'): если игрок в мире и никакой экран не открыт,
    // а играет не трек зоны — возвращаем трек зоны.
    let zoneWatchTimer = null, lastZoneSeen = -2;
    function followZone() {
        const z = zoneIndex();
        if (z < 0) { lastZoneSeen = -2; return; }
        const want = moodForZone(z);
        if (z !== lastZoneSeen) { lastZoneSeen = z; mood(want); return; }
        if (want !== curMood && !screenMusicBusy()) mood(want);
    }
    function startZoneWatch() {
        if (zoneWatchTimer) return;
        zoneWatchTimer = setInterval(() => { try { followZone(); } catch (e) { /* ignore */ } }, 500);
    }

    let curMood = 'off';
    let pendingMood = null;      // трек, который надо поднять после разблокировки
    let curSlot = null;          // { name, song, gain, ... stepIdx, nextStep }
    let schedTimer = null;       // setInterval секвенсора

    // Кривая овердрайва (общая): одна на приложение, drive задаёт предусиление.
    let distBuf = null;
    function distCurve() {
        if (distBuf) return distBuf;
        const n = 1024;
        distBuf = new Float32Array(n);
        const k = Math.tanh(3.2);
        for (let i = 0; i < n; i++) {
            const x = (i / (n - 1)) * 2 - 1;
            distBuf[i] = Math.tanh(x * 3.2) / k;
        }
        return distBuf;
    }

    // ===== Треки: ОДИН активный =====
    // У каждой песни СВОЙ выходной тракт: voices → фильтр cut → [овердрайв] →
    // gain → musicGain, плюс send в реверб-шину. При переключении старый тракт
    // гасится и отключается целиком — песни физически не могут наслаиваться.
    function makeSlot(name, s) {
        const gain = ctx.createGain();
        gain.gain.value = 0.0001;
        const filter = ctx.createBiquadFilter();
        filter.type = 'lowpass';
        filter.frequency.value = s.cut || 2400;
        filter.Q.value = 0.6;
        // Овердрайв только у боевых тем (Chaos): без него сай звучит «цифрово».
        let distNodes = null;
        const drive = clamp(s.dist || 0, 0, 1);
        if (drive > 0 && ctx.createWaveShaper) {
            const pre = ctx.createGain();
            pre.gain.value = 1 + 5 * drive;
            const sh = ctx.createWaveShaper();
            sh.curve = distCurve();
            sh.oversample = '2x';
            // post — компенсация громкости: дистошн поднимает уровень в разы,
            // боевая тема не должна орать громче спокойной.
            const post = ctx.createGain();
            post.gain.value = 1 / (1 + 2.5 * drive);
            distNodes = { pre: pre, sh: sh, post: post };
            filter.connect(pre);
            pre.connect(sh);
            sh.connect(post);
            post.connect(gain);
        } else {
            filter.connect(gain);
        }
        gain.connect(musicGain);
        // Реверб-send: доля сухого сигнала трека в общий зал.
        let revSend = null;
        const rev = clamp(s.rev != null ? s.rev : 0.25, 0, 1);
        if (rev > 0 && revConv) {
            revSend = ctx.createGain();
            revSend.gain.value = rev;
            gain.connect(revSend);
            revSend.connect(revConv);
        }
        // Эхо — только у «просторных» тем (низкие ступени, меню, галерея).
        // Боевым трекам оно не нужно: важна плотность, а не воздух.
        let delay = null, delayGain = null, feedback = null;
        const echo = s.echo != null ? s.echo : 0.2;
        if (echo > 0) {
            delay = ctx.createDelay(1.2);
            delay.delayTime.value = s.delay || 0.29;
            feedback = ctx.createGain();
            feedback.gain.value = 0.24;
            delayGain = ctx.createGain();
            delayGain.gain.value = echo;
            filter.connect(delay);
            delay.connect(feedback);
            feedback.connect(delay);
            delay.connect(delayGain);
            delayGain.connect(gain);
        }
        // Таймлайн секций: накопительные границы тактов (считается один раз).
        let totalBars = 0;
        const starts = [];
        const secList = (s.sections && s.sections.length) ? s.sections : [sec(4, {})];
        for (const sc of secList) {
            starts.push({ start: totalBars, sec: sc });
            totalBars += Math.max(1, sc.bars | 0);
        }
        const slot = {
            name: name, song: s, gain: gain, filter: filter,
            delay: delay, delayGain: delayGain, feedback: feedback,
            dist: distNodes, revSend: revSend,
            starts: starts, totalBars: totalBars,
            stepIdx: 0, nextStep: 0, dead: false,
        };
        liveSlots.add(slot);
        return slot;
    }

    // Живые тракты (с ними же сверяемся в debug(): одновременно не больше одного).
    const liveSlots = new Set();

    // Погасить и физически выкинуть тракт трека (вместе с его эхом и ревербом).
    function killSlot(slot, fade) {
        if (!slot || slot.dead) return;
        slot.dead = true;
        liveSlots.delete(slot);
        try {
            const t = now();
            const gv = Math.max(0.0001, slot.gain.gain.value);
            slot.gain.gain.cancelScheduledValues(t);
            slot.gain.gain.setValueAtTime(gv, t);
            slot.gain.gain.linearRampToValueAtTime(0.0001, t + fade);
        } catch (e) { /* ignore */ }
        setTimeout(() => {
            try {
                if (slot.delay) { slot.delay.disconnect(); slot.delayGain.disconnect(); slot.feedback.disconnect(); }
                if (slot.revSend) slot.revSend.disconnect();
                if (slot.dist) {
                    slot.dist.pre.disconnect();
                    slot.dist.sh.disconnect();
                    slot.dist.post.disconnect();
                }
                slot.filter.disconnect();
                slot.gain.disconnect();
            } catch (e) { /* ignore */ }
        }, Math.ceil(fade * 1000) + 90);
    }

    // Секвенсор: планируем на 0.15с вперёд — таймер может «дрожать», звук нет.
    function pump() {
        const slot = curSlot;
        if (!ctx || !slot || slot.dead) return;
        const stepDur = (60 / slot.song.bpm) / 4;      // длительность 16-го шага
        if (!slot.nextStep) slot.nextStep = now() + 0.08;
        while (slot.nextStep < now() + 0.15) {
            scheduleStep(slot, slot.nextStep, slot.stepIdx);
            slot.nextStep += stepDur;
            slot.stepIdx++;
        }
    }

    // Секция для такта bar (таймлайн уже посчитан в makeSlot).
    function sectionAt(slot, bar) {
        const list = slot.starts;
        for (let i = list.length - 1; i >= 0; i--) {
            if (bar >= list[i].start) {
                return { sec: list[i].sec, barIn: bar - list[i].start, idx: i };
            }
        }
        return { sec: list[0].sec, barIn: 0, idx: 0 };
    }

    // «Человеческий» тайминг: неровность ±мс (не для бочки — она держит грув).
    function hum(t, ms) { return t + rnd(-ms, ms) / 1000; }

    // Филл на последнем такте секции: подкат к смене (снейр с нарастанием).
    function fillAt(t, s16, kind, vel, dest) {
        if (kind === 'big') {
            if (s16 >= 12) snareAt(hum(t, 3), vel, 0.6 + (s16 - 12) * 0.35, dest);
        } else if (kind === 'light') {
            if (s16 === 13 || s16 === 15) snareAt(hum(t, 3), vel, 0.9, dest);
            if (s16 === 14) hatAt(t, vel, true, dest);
        }
    }

    function scheduleStep(slot, t0, idx) {
        const s = slot.song;
        const st = 16;
        const bar = Math.floor(idx / st) % slot.totalBars;
        const pos = sectionAt(slot, bar);
        const sec_ = pos.sec;
        const s16 = idx % st;
        const dest = slot.filter;              // выход ИМЕННО этой песни
        const vel = sec_.vel != null ? sec_.vel : 1;
        const prog = sec_.prog || s.prog || [0];
        const root = s.key + prog[pos.barIn % prog.length];
        const isLastBar = pos.barIn === sec_.bars - 1;

        // ---------- барабаны ----------
        const kick = sec_.kick || s.kick || [];
        const snare = sec_.snare || s.snare || [];
        const hat = sec_.hat || s.hat || [];
        if (kick.indexOf(s16) >= 0) kickAt(t0, vel, dest);
        if (snare.indexOf(s16) >= 0) snareAt(hum(t0, 3), vel, 1, dest);
        if (hat.indexOf(s16) >= 0) hatAt(hum(t0, 7), vel, false, dest);
        if (isLastBar && sec_.fill) fillAt(t0, s16, sec_.fill, vel, dest);
        if (sec_.crash && s16 === 0) crashAt(t0, vel, dest);

        // ---------- бас ----------
        const bass = sec_.bass || s.bass || [];
        const b = bass[s16];
        if (b != null) {
            bassAt(root + b, hum(t0, 4), sec_.bassDur || s.bassDur || 0.2, {
                wave: s.bassWave || 'sawtooth', gain: (s.bassGain || 0.06) * vel *
                    (s16 % 4 === 0 ? 1.1 : rnd(0.88, 1)),
                cut: s.bassCut || 520, dest: dest,
            });
        }
        // ---------- арпеджио/лид ----------
        const arp = sec_.arp || s.arp;
        const a = arp ? arp[s16] : null;
        if (a != null) {
            const wave = sec_.arpWave || s.arpWave || 'triangle';
            const ag = (sec_.arpGain || s.arpGain || 0.03) * vel *
                (s16 % 4 === 0 ? 1.12 : rnd(0.85, 1));
            const atk = sec_.arpAttack || s.arpAttack || 0.004;
            if (wave === 'bell') {
                bellAt(root + a + (s.arpOct || 12), hum(t0, 6),
                    sec_.arpDur || s.arpDur || 0.15,
                    { gain: ag, attack: atk, pan: -0.16, dest: dest });
            } else {
                voice(midiHz(root + a + (s.arpOct || 12)), hum(t0, 6),
                    sec_.arpDur || s.arpDur || 0.13, {
                        wave: wave, gain: ag, attack: atk, detune: rnd(-6, 6),
                        pan: -0.16, dest: dest,
                    });
            }
        }
        // ---------- контр-мелодия (второй голос) ----------
        const lead = sec_.lead || s.lead;
        const l = lead ? lead[s16] : null;
        if (l != null) {
            voice(midiHz(root + l + (s.leadOct || 24)), hum(t0, 8),
                sec_.leadDur || s.leadDur || 0.3, {
                    wave: s.leadWave || 'sine', gain: (s.leadGain || 0.018) * vel,
                    attack: 0.02, vib: 5.2, vibAmt: 9, pan: 0.2, dest: dest,
                });
        }
        // ---------- пауэр-аккорд (боевые ступени) ----------
        const stab = sec_.stab || s.stab;
        if (stab && stab.indexOf(s16) >= 0) {
            const sg = (s.stabGain || 1) * vel;
            [0, 7, 12, 19].forEach((iv, k) =>
                voice(midiHz(root + iv - 12), t0, 0.3, {
                    wave: 'sawtooth', gain: 0.028 * sg * (k ? 0.7 : 1),
                    attack: 0.004, filter: 'lowpass', cutoff: 1100, dest: dest,
                }));
        }
        // ---------- пэд: длинный аккорд в начале такта ----------
        const padOn = (sec_.pad != null) ? sec_.pad : s.pad;
        if (padOn && s16 === 0) {
            padAt(root, t0, 2.6, {
                wave: s.padWave || 'triangle', gain: (s.padGain || 0.02) * vel,
                cutoff: 1100, ivs: s.padIv || [0, 7, 12, 15], dest: dest,
            });
        }
    }

    function startScheduler() {
        if (schedTimer || !ctx) return;
        schedTimer = setInterval(() => { try { pump(); } catch (e) { /* ignore */ } }, 25);
    }
    function stopScheduler() {
        if (schedTimer) { clearInterval(schedTimer); schedTimer = null; }
    }

    // mood — сменить трек. Старый тракт гасится за 0.18с и выкидывается целиком:
    // музыка НИКОГДА не накладывается сама на себя. 'off' — тишина.
    function mood(name) {
        if (!enabled || !AC) return;
        if (name === curMood) return;
        if (name !== 'off' && !SONGS[name]) return;    // неизвестный трек — не глушим музыку
        // Контекста ещё нет (ждем жеста) — запоминаем трек и включим после unlock.
        if (name !== 'off' && !ctx) { pendingMood = name; return; }
        curMood = name;
        if (!ctx) return;
        const old = curSlot;
        curSlot = null;
        if (old) killSlot(old, 0.18);
        if (name === 'off') { stopScheduler(); return; }
        curSlot = makeSlot(name, SONGS[name]);
        const t = now();
        try {
            curSlot.gain.gain.cancelScheduledValues(t);
            curSlot.gain.gain.setValueAtTime(0.0001, t);
            curSlot.gain.gain.linearRampToValueAtTime(1, t + 0.42);
        } catch (e) { /* ignore */ }
        startScheduler();
    }

    // onScreen — единая точка смены музыки по экранам (зовётся из showScreen).
    // У каждого экрана своё настроение: меню, галерея (музей), реалм, витрина
    // Skins (торговая) и редактор аватара (эмбиент). Мир — по ступени биома
    // (followZone ниже).
    function onScreen(name) {
        if (name === 'gallery') mood('gallery');
        else if (name === 'realm') mood('realm');
        else if (name === 'shop') mood('shop');
        else if (name === 'avatar') mood('avatar');
        else if (name === 'menu' || name === 'login' || name === 'register' ||
                 name === 'color') mood('menu');
    }

    // ------------------------------------------------------ хуки игровых систем
    // Бюджет ударов: 18/сек; повторные удары в одном кадре (<28мс) сливаются —
    // у игрока 10 петалей, звук не должен превращаться в кашу.
    let hitTokens = 18, hitRefill = performance.now(), lastHitAt = 0;
    function hitAllowed(nowMs) {
        const dt = Math.max(0, nowMs - hitRefill);
        if (dt > 0) { hitRefill = nowMs; hitTokens = Math.min(18, hitTokens + dt * 0.018); }
        if (hitTokens < 1) return false;
        if (nowMs - lastHitAt < 28) return false;
        hitTokens -= 1;
        lastHitAt = nowMs;
        return true;
    }

    // Позиция моего игрока (для громкости/панорамы) — аккуратно, без падений.
    function myPos() {
        try {
            if (typeof actors === 'undefined' || !window.myId) return null;
            const a = actors[window.myId];
            if (!a) return null;
            if (a.sprite && a.sprite.parent) return { x: a.sprite.x, y: a.sprite.y };
            if (a.targetX != null) return { x: a.targetX, y: a.targetY };
        } catch (e) { /* ignore */ }
        return null;
    }

    function petalHitAtMob(a, m, hitR, dmg) {
        if (!enabled) return;
        const ms = performance.now();
        if (!hitAllowed(ms)) return;
        const maxHp = (m && m.maxHp > 0) ? m.maxHp : Math.max(1, (dmg || 0) * 4);
        const power = clamp((dmg || 0) / maxHp / 0.1, 0.15, 1);
        let pan = 0, gain = 1;
        const me = myPos();
        const mx = (a && a.c && a.c.x != null) ? a.c.x : (m && m.x);
        const my = (a && a.c && a.c.y != null) ? a.c.y : (m && m.y);
        if (me && mx != null) {
            const dx = mx - me.x, dy = my - me.y;
            const dist = Math.hypot(dx, dy);
            if (dist > 1500) return;                    // далеко — не слышно
            gain = clamp(1 - dist / 1500, 0.1, 1);
            pan = clamp(dx / 760, -1, 1);
        }
        playSfx('hit', { power: power, ri: hitR || 0, pan: pan, gain: gain });
    }

    // Обёртки: mobHitFX (удар петали по мобу) и updateHP (мой урон).
    function installHooks() {
        try {
            const origHit = window.mobHitFX;
            if (typeof origHit === 'function' && !origHit.__audioHooked) {
                const wrapped = function (a, m, hitR, dmg) {
                    try { origHit(a, m, hitR, dmg); } finally {
                        try { petalHitAtMob(a, m, hitR, dmg); } catch (e) { /* ignore */ }
                    }
                };
                wrapped.__audioHooked = true;
                window.mobHitFX = wrapped;
            }
        } catch (e) { /* ignore */ }

        try {
            const origHP = window.updateHP;
            if (typeof origHP === 'function' && !origHP.__audioHooked) {
                let prev = null;
                const wrapped = function (hp, maxHp, shield, maxShield) {
                    const r = origHP(hp, maxHp, shield, maxShield);
                    try {
                        if (prev != null && hp < prev - 0.5) {
                            const p = clamp((prev - hp) / Math.max(1, maxHp) / 0.12, 0.15, 1);
                            playSfx('hurt', { power: p });
                            // Вибрация геймпада на полученном уроне (gamepad.js).
                            if (window.GamepadRumble) window.GamepadRumble.pulse(90 + 140 * p, p);
                        }
                        prev = hp;
                    } catch (e) { /* ignore */ }
                    return r;
                };
                wrapped.__audioHooked = true;
                window.updateHP = wrapped;
            }
        } catch (e) { /* ignore */ }

        try {
            const origScreen = window.showScreen;
            if (typeof origScreen === 'function' && !origScreen.__audioHooked) {
                const wrapped = function (name) {
                    const r = origScreen.apply(this, arguments);
                    try { onScreen(name); } catch (e) { /* ignore */ }
                    return r;
                };
                wrapped.__audioHooked = true;
                window.showScreen = wrapped;
            }
        } catch (e) { /* ignore */ }
    }

    // ---------------------------------------------------------------- события
    // Первый жест — разблокировка звука (браузерное требование к автоплею).
    ['pointerdown', 'keydown', 'touchstart'].forEach((ev) => {
        window.addEventListener(ev, unlock, { once: true, capture: true });
    });
    // Вкладка скрыта — музыка приглушается (не крутим звук в фоне).
    document.addEventListener('visibilitychange', () => {
        if (!ctx || !musicDuck) return;
        try {
            musicDuck.gain.setTargetAtTime(document.hidden ? 0.0001 : 1, now(), 0.2);
        } catch (e) { /* ignore */ }
    });

    // Публичный API.
    const api = {
        unlock: unlock,
        sfx: playSfx,
        ui: playUi,
        mood: mood,
        currentMood: () => curMood,
        onScreen: onScreen,
        setMix: function (partial) {
            if (!partial) return;
            if (partial.master != null) mix.master = clamp(+partial.master || 0, 0, 1);
            if (partial.music != null) mix.music = clamp(+partial.music || 0, 0, 1);
            if (partial.sfx != null) mix.sfx = clamp(+partial.sfx || 0, 0, 1);
            if (partial.ui != null) mix.ui = clamp(+partial.ui || 0, 0, 1);
            applyMix();
        },
        mix: function () { return { master: mix.master, music: mix.music, sfx: mix.sfx, ui: mix.ui }; },
        setEnabled: function (on) {
            enabled = !!on;
            if (!enabled) {
                try { if (ctx && ctx.suspend) ctx.suspend(); } catch (e) {}
                stopScheduler();
                if (curSlot) { killSlot(curSlot, 0.05); curSlot = null; }
                curMood = 'off';
            } else {
                unlock();
                if (pendingMood) { const m = pendingMood; pendingMood = null; mood(m); }
            }
            applyMix();
        },
        enabled: () => enabled,
        // Готов ли движок (контекст живёт) — для индикаторов в UI.
        ready: () => !!ctx && !!(ctx.state !== 'suspended'),
        // Трек текущей зоны (вручную — на случай перехода без опроса).
        zone: () => mood(moodForZone(zoneIndex())),
        songs: () => Object.keys(SONGS),
        // Диагностика: живых трактов должно быть ≤ 1 (иначе музыка слоится).
        debug: () => ({ mood: curMood, live: liveSlots.size, pending: pendingMood }),
    };
    window.GameAudio = api;
    // Короткие алиасы для магазина/гардероба/других модулей.
    window.Sfx = { play: playSfx, ui: playUi };

    // Опрос зоны живёт всегда: пока игрок в меню, myZone весь < 0 и опрос
    // просто молчит (музыка экранов его не боится).
    startZoneWatch();

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', installHooks);
    } else {
        installHooks();
    }
})();
