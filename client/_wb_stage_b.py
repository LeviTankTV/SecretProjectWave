# -*- coding: utf-8 -*-
import io

# ===== 1. HTML: recap block inside #death =====
p = 'index.html'
s = io.open(p, encoding='utf-8').read()
old = '''    <div id="death">
        <h2 id="deathTitle">You Died!</h2>
        <div id="deathSub" style="color:#9fb2d0;font-size:14px;display:none;"></div>
        <div id="deathWaveLine"></div>
        <div class="btn-group">
            <button id="respawnBtn">Respawn</button>
            <button id="menuBtn">Return to Main Menu</button>
        </div>
    </div>'''
new = '''    <div id="death">
        <!-- WAVEBORN рекап забега (gameover) -->
        <div id="wbRecap" style="display:none;">
            <div class="wb-recap-label" data-wb="death.wiped">PARTY WIPED</div>
            <div class="wb-recap-wave"><span id="wbRecapWave">0</span></div>
            <div class="wb-recap-word" data-wb="death.wave">WAVE</div>
            <div class="wb-recap-stats">
                <div class="wb-rs"><span id="wbRecapKills">0</span><small data-wb="death.kills">kills</small></div>
                <div class="wb-rs"><span id="wbRecapTime">0:00</span><small data-wb="death.time">run time</small></div>
                <div class="wb-rs"><span id="wbRecapBest">0</span><small data-wb="death.best">best</small></div>
            </div>
            <div class="btn-group">
                <button id="wbRetryBtn" data-wb="death.retry">RETRY</button>
                <button id="menuBtn2">Return to Main Menu</button>
            </div>
        </div>
        <!-- Обычная смерть (респавн на старте волны) -->
        <div id="wbDeathSimple">
            <h2 id="deathTitle">You Died!</h2>
            <div id="deathSub" style="color:#9fb2d0;font-size:14px;display:none;"></div>
            <div id="deathWaveLine"></div>
            <div class="btn-group">
                <button id="respawnBtn">Respawn</button>
                <button id="menuBtn">Return to Main Menu</button>
            </div>
        </div>
    </div>'''
assert old in s, 'death block'
s = s.replace(old, new, 1)

# menuBtn2 -> дублируем обработчик menuBtn (привязка ниже по id menuBtn).
# Проще: обработчик menuBtn в JS кликает по id; добавим в HTML onclick-алиас.
s = s.replace('id="menuBtn2"', 'id="menuBtn2" onclick="document.getElementById("menuBtn").click()"', 1)

# Add banner recap CSS (THEME 3.1)
anchor = '</head>'
css = '''
    <style>
        /* ---------- THEME 3.1: баннер-гигант + рекап смерти ---------- */
        /* Баннер волны: огромный номер по центру, без подложки */
        #waveBanner {
            background: none !important; border: none !important;
            box-shadow: none !important; backdrop-filter: none !important;
            border-radius: 0 !important;
            top: 18% !important;
        }
        #waveBannerMain {
            font-family: var(--wb-font) !important;
            font-size: clamp(64px, 14vh, 128px) !important;
            font-weight: 700 !important; letter-spacing: 6px !important;
            color: #fff !important;
            text-shadow:
                0 0 8px rgba(0,240,255,0.9), 0 0 30px rgba(0,240,255,0.5),
                3px 0 0 rgba(255,46,136,0.5), -3px 0 0 rgba(0,240,255,0.5) !important;
            animation: wbBannerIn .45s cubic-bezier(.2,.9,.3,1.1), wbWavePulse 2.4s ease-in-out .45s infinite;
        }
        #waveBanner .wave-sub {
            font-family: var(--wb-mono) !important;
            font-size: 15px !important; color: var(--wb-cyan) !important;
            letter-spacing: 5px !important; text-transform: uppercase !important;
        }
        @keyframes wbBannerIn {
            from { transform: scale(1.35); opacity: 0; filter: blur(6px); }
            to   { transform: scale(1); opacity: 1; filter: none; }
        }
        @keyframes wbWavePulse {
            0%, 100% { text-shadow: 0 0 8px rgba(0,240,255,0.9), 0 0 30px rgba(0,240,255,0.5), 3px 0 0 rgba(255,46,136,0.5), -3px 0 0 rgba(0,240,255,0.5); }
            50% { text-shadow: 0 0 14px rgba(0,240,255,1), 0 0 46px rgba(0,240,255,0.7), 2px 0 0 rgba(255,46,136,0.5), -2px 0 0 rgba(0,240,255,0.5); }
        }
        /* Рекап смерти WAVEBORN */
        #wbRecap { display: flex; flex-direction: column; align-items: center; gap: 10px; }
        .wb-recap-label {
            font-family: var(--wb-mono); font-size: 14px; letter-spacing: 6px;
            color: var(--wb-magenta); text-transform: uppercase;
        }
        .wb-recap-wave span {
            font-family: var(--wb-font); font-weight: 700;
            font-size: clamp(90px, 20vh, 170px); line-height: 1;
            color: #fff;
            text-shadow: 0 0 10px rgba(0,240,255,0.9), 0 0 40px rgba(0,240,255,0.45),
                4px 0 0 rgba(255,46,136,0.45), -4px 0 0 rgba(0,240,255,0.45);
        }
        .wb-recap-word {
            font-family: var(--wb-font); font-size: 15px; font-weight: 600;
            letter-spacing: 10px; color: var(--wb-cyan); text-transform: uppercase;
            margin-top: -8px;
        }
        .wb-recap-stats { display: flex; gap: 42px; margin: 14px 0 6px; }
        .wb-rs { display: flex; flex-direction: column; align-items: center; gap: 2px; }
        .wb-rs span {
            font-family: var(--wb-font); font-weight: 700; font-size: 30px; color: #fff;
            text-shadow: 0 0 12px rgba(0,240,255,0.4);
        }
        .wb-rs small {
            font-family: var(--wb-mono); font-size: 11px; letter-spacing: 2px;
            color: var(--wb-dim); text-transform: uppercase;
        }
        #wbRecap .btn-group button, #wbDeathSimple .btn-group button {
            font-family: var(--wb-font) !important;
        }
        #wbRetryBtn {
            background: linear-gradient(180deg, rgba(0,240,255,0.16), rgba(0,240,255,0.05)) !important;
            border: 1px solid var(--wb-cyan) !important; color: var(--wb-cyan) !important;
            font-weight: 700; letter-spacing: 4px;
            clip-path: polygon(0 0, calc(100% - 12px) 0, 100% 12px, 100% 100%, 12px 100%, 0 calc(100% - 12px));
        }
        #wbRetryBtn:hover { background: var(--wb-cyan) !important; color: #02131a !important; box-shadow: 0 0 30px rgba(0,240,255,0.5) !important; }
        /* Экран смерти в стиле WAVEBORN */
        #death {
            background:
                radial-gradient(60% 45% at 50% 42%, rgba(255,46,136,0.10), transparent 70%),
                radial-gradient(40% 30% at 50% 45%, rgba(0,240,255,0.06), transparent 70%),
                rgba(2,4,9,0.93) !important;
            backdrop-filter: blur(14px); -webkit-backdrop-filter: blur(14px);
        }
        #wbDeathSimple h2 {
            background: none !important; -webkit-background-clip: initial !important; background-clip: initial !important;
            color: var(--wb-magenta) !important;
            text-shadow: 0 0 20px rgba(255,46,136,0.5) !important;
            font-family: var(--wb-font) !important; letter-spacing: 8px !important;
        }
    </style>
'''
i = s.find(anchor)
s = s[:i] + css + '\n' + s[i:]
io.open(p, 'w', encoding='utf-8', newline='').write(s)
print('HTML OK')

# ===== 2. i18n keys for death (main.js) =====
p = 'js/main.js'
s = io.open(p, encoding='utf-8').read()
old = "    'lobby.noGames':        { en: 'No open games — create your own!', ru: 'Открытых игр нет — создай свою!' },\n};"
new = """    'lobby.noGames':        { en: 'No open games — create your own!', ru: 'Открытых игр нет — создай свою!' },
    'death.wiped':          { en: 'PARTY WIPED', ru: 'ПАТИ ВЫРЕЗАНО' },
    'death.wave':           { en: 'WAVE', ru: 'ВОЛНА' },
    'death.kills':          { en: 'kills', ru: 'киллов' },
    'death.time':           { en: 'run time', ru: 'время' },
    'death.best':           { en: 'best', ru: 'рекорд' },
    'death.retry':          { en: 'RETRY', ru: 'ЗАНОВО' },
    'death.newBest':        { en: 'NEW BEST!', ru: 'НОВЫЙ РЕКОРД!' },
    'wave.cleared':         { en: 'cleared', ru: 'зачищена' },
    'wave.next':            { en: 'Next wave soon', ru: 'Следующая волна скоро' },
    'wave.boss':            { en: 'BOSS — a ladder boss has awakened', ru: 'БОСС — проснулся босс лестницы' },
    'wave.elite':           { en: 'ELITE — every mob is a variant', ru: 'ЭЛИТА — каждый моб с подвидом' },
    'wave.swarm':           { en: 'SWARM — many small fast mobs', ru: 'РОЙ — много мелких быстрых мобов' },
    'wave.tanks':           { en: 'TANKS — heavy high-rarity mobs', ru: 'ТАНКИ — тяжёлые редкие мобы' },
    'wave.hunters':         { en: 'HUNTERS — aggressive chasers', ru: 'ОХОТНИКИ — агрессивные преследователи' },
};"""
assert old in s, 'i18n tail'
s = s.replace(old, new, 1)
io.open(p, 'w', encoding='utf-8', newline='').write(s)
print('main.js keys OK')
