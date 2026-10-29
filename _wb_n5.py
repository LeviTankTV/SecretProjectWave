# -*- coding: utf-8 -*-
import io

# ===== 1. СЕРВЕР: died с данными убийцы (карточка убийцы в рекапе) =====
p = 'server/room.go'
s = io.open(p, encoding='utf-8').read()
old = '''				if p.State.HP <= 0 {
					p.State.HP = 0
					p.Dead = true
					if cl := r.clients[p.State.ID]; cl != nil {
						cl.deliver([]byte(`{"type":"died"}`))
					}
				}
				break
			}
		}
	}'''
new = '''				if p.State.HP <= 0 {
					p.State.HP = 0
					p.Dead = true
					if cl := r.clients[p.State.ID]; cl != nil {
						// WAVEBORN: died с данными убийцы (карточка в рекапе).
						died, _ := json.Marshal(map[string]interface{}{
							"type": "died",
							"killerType": m.Type, "killerRarity": m.Rarity,
							"killerVariant": m.Variant, "killerName": mobTypes[m.Type].Name,
						})
						cl.deliver(died)
					}
				}
				break
			}
		}
	}'''
assert old in s, 'died killer'
s = s.replace(old, new, 1)

# ===== 2. СЕРВЕР: mobDied с убийцей (мультикиллы) =====
old = '''	died := map[string]interface{}{
		"type": "mobDied", "id": m.ID,
		"x": m.X, "y": m.Y, "size": m.Radius,
		"shape": mt.Shape, "color": diedColor,
	}'''
new = '''	died := map[string]interface{}{
		"type": "mobDied", "id": m.ID,
		"x": m.X, "y": m.Y, "size": m.Radius,
		"shape": mt.Shape, "color": diedColor,
		"killer": m.KillerID, // WAVEBORN: для мультикилл-объявлений
	}'''
assert old in s, 'mobDied killer'
s = s.replace(old, new, 1)
io.open(p, 'w', encoding='utf-8', newline='').write(s)
print('server OK')

# ===== 3. КЛИЕНТ: кинематографический пакет =====
p = 'client/js/main.js'
s = io.open(p, encoding='utf-8').read()

# 3a. died: запомнить убийцу
old = '''            case 'died':
                sfx('death');
                showDeath();
                break;'''
new = '''            case 'died':
                sfx('death');
                try {
                    window.__wbKiller = (msg.killerType) ? {
                        type: msg.killerType, name: msg.killerName || msg.killerType,
                        rarity: msg.killerRarity || 0, variant: msg.killerVariant || '',
                    } : null;
                } catch (e) {}
                showDeath();
                break;'''
assert old in s, 'died killer'
s = s.replace(old, new, 1)

# 3b. mobDied: мультикиллы по killer
old = '''            case 'mobDied':
                sfx('mobKill');
                if (msg.color === 0xffd700) {
                    const live2 = mobActors[msg.id];
                    wbShowDropAlert(null, -1, true, msg, live2 ? live2.mname || '' : '');
                    wbJingle(7);
                }
                spawnMobFade(msg);
                break;'''
new = '''            case 'mobDied': {
                sfx('mobKill');
                if (msg.color === 0xffd700) {
                    const live2 = mobActors[msg.id];
                    wbShowDropAlert(null, -1, true, msg, live2 ? live2.mname || '' : '');
                    wbJingle(7);
                }
                // WAVEBORN: серии убийств (2с окно) — DOUBLE/TRIPLE/RAMPAGE.
                if (msg.killer && msg.killer === myId) {
                    const now = performance.now();
                    const w = (window.__wbStreak = window.__wbStreak || []);
                    while (w.length && now - w[0] > 2000) w.shift();
                    w.push(now);
                    if (w.length >= 2 && w.length <= 4) {
                        const names = { 2: 'DOUBLE KILL!', 3: 'TRIPLE KILL!', 4: 'RAMPAGE!' };
                        wbShowMultikill(names[w.length]);
                        wbJingle(5);
                    }
                }
                spawnMobFade(msg);
                break;
            }'''
assert old in s, 'multikill'
s = s.replace(old, new, 1)

# 3c. wbShowMultikill + wbBossIntro + wbZoneTitle (DOM-функции)
old = '''// ===== WAVEBORN: крупный алерт редкого дропа (RICHNESS №17) ====='''
new = '''// ===== WAVEBORN кинематограф (Н5): мультикилл / босс-интро / титул зоны =====
function wbCineShow(id, html, ms) {
    let box = document.getElementById(id);
    if (!box) {
        box = document.createElement('div');
        box.id = id;
        document.body.appendChild(box);
    }
    box.innerHTML = html;
    box.style.display = 'flex';
    clearTimeout(box._t);
    box._t = setTimeout(() => { box.style.display = 'none'; }, ms || 1500);
}
function wbShowMultikill(text) {
    wbCineShow('wbMultikill',
        '<div class="wb-mk">' + text + '</div>', 1300);
}
// Босс-интро: парс superSpawn "A Chaos <name> has spawned!".
function wbShowBossIntro(tierName, mobName) {
    const col = rarityColors[Math.min(rarityNames.indexOf(tierName), rarityColors.length - 1)] || '#ff2d95';
    wbCineShow('wbBossIntro',
        '<div class="wb-bi-label" data-i18n-skip>— BOSS —</div>' +
        '<div class="wb-bi-name" style="color:' + col + ';text-shadow:0 0 24px ' + col + '88">' +
        escapeHtml((tierName + ' ' + mobName).toUpperCase()) + '</div>' +
        '<div class="wb-bi-sub">has awakened</div>', 2200);
    wbShake(10, 0.5);
}
// Титул зоны при старте матча (как «Garden» у florr).
function wbShowZoneTitle(zoneIdx) {
    const z = (typeof zones !== 'undefined' && zones[zoneIdx]) ? zones[zoneIdx] : null;
    if (!z) return;
    const col = rarityNum ? intToHex(z.color) : '#ffffff';
    wbCineShow('wbZoneTitle',
        '<div class="wb-zt">' + escapeHtml(z.name || '') + '</div>', 2000);
}

// ===== WAVEBORN: крупный алерт редкого дропа (RICHNESS №17) ====='''
assert old in s, 'cine fns'
s = s.replace(old, new, 1)

# 3d. rarityNames в клиенте есть (rarities.js) — ок. Hook sys msgs для босс-интро:
old = '''            case 'sys':
                addSystemToast(msg.text || '', msg.kind || '');
                break;'''
new = '''            case 'sys':
                addSystemToast(msg.text || '', msg.kind || '');
                // WAVEBORN: босс-интро из анонса супо-спавна.
                try {
                    const mm = /^(A|An)\\s+(\\w+)\\s+(.+?)\\s+has spawned!$/.exec(msg.text || '');
                    if (mm && rarityNames.includes(mm[2])) wbShowBossIntro(mm[2], mm[3]);
                } catch (e) {}
                break;'''
assert old in s, 'sys intro'
s = s.replace(old, new, 1)

# 3e. Титул зоны на старте волны 1 (handleWaveMsg в lobby.js — зона из state)
p = 'client/js/lobby.js'
s2 = io.open(p, encoding='utf-8').read()
old = '''            showWaveBanner('Wave ' + msg.num, sub, 3500);
            window.__wbBossGlow = (msg.special === 'boss');'''
new = '''            showWaveBanner('Wave ' + msg.num, sub, 3500);
            window.__wbBossGlow = (msg.special === 'boss');
            // WAVEBORN: титул зоны на первой волне матча.
            if (msg.num === 1 && typeof wbShowZoneTitle === 'function') {
                try { wbShowZoneTitle(typeof myZone === 'number' ? myZone : 0); } catch (e) {}
            }'''
assert old in s2; s2 = s2.replace(old, new, 1)

# 3f. Рекап: карточка убийцы
old = '''                const lbl = recap.querySelector('.wb-recap-label');
                if (lbl && window.tr) {
                    lbl.textContent = isNewBest ? window.tr('death.newBest') : window.tr('death.wiped');
                }
            }
            break;
        }
    }
}'''
new = '''                const lbl = recap.querySelector('.wb-recap-label');
                if (lbl && window.tr) {
                    lbl.textContent = isNewBest ? window.tr('death.newBest') : window.tr('death.wiped');
                }
                // WAVEBORN: карточка убийцы (кто добил).
                const kb = recap.querySelector('.wb-killer');
                if (kb) {
                    const k = window.__wbKiller;
                    if (k && k.type) {
                        const kcol = rarityColors[Math.min(k.rarity || 0, rarityColors.length - 1)];
                        kb.innerHTML = '<span class="wb-killer-by">Slain by</span>' +
                            '<b style="color:' + kcol + '">' + escapeHtml((k.variant ? k.variant + ' ' : '') + k.name) + '</b>' +
                            '<span class="wb-killer-rar" style="border-color:' + kcol + ';color:' + kcol + '">' +
                            (rarityName ? rarityName(k.rarity) : k.rarity) + '</span>';
                        kb.style.display = 'flex';
                    } else {
                        kb.style.display = 'none';
                    }
                }
            }
            break;
        }
    }
}'''
assert old in s2; s2 = s2.replace(old, new, 1)

# сброс __wbKiller на новой волне
old = '''    // WAVEBORN: рекап прячем, простой экран возвращаем.
    const rec = document.getElementById('wbRecap');'''
new = '''    // WAVEBORN: рекап прячем, простой экран возвращаем.
    window.__wbKiller = null;
    const rec = document.getElementById('wbRecap');'''
assert old in s2; s2 = s2.replace(old, new, 1)
io.open(p, 'w', encoding='utf-8', newline='').write(s2)
print('client OK')

# ===== 4. index.html: killer-слот в рекапе + CSS кинематографа =====
p = 'index.html'
s = io.open(p, encoding='utf-8').read()
old = '''            <div class="wb-recap-stats">
                <div class="wb-rs"><span id="wbRecapKills">0</span><small data-wb="death.kills">kills</small></div>
                <div class="wb-rs"><span id="wbRecapTime">0:00</span><small data-wb="death.time">run time</small></div>
                <div class="wb-rs"><span id="wbRecapBest">0</span><small data-wb="death.best">best</small></div>
            </div>'''
new = '''            <div class="wb-recap-stats">
                <div class="wb-rs"><span id="wbRecapKills">0</span><small data-wb="death.kills">kills</small></div>
                <div class="wb-rs"><span id="wbRecapTime">0:00</span><small data-wb="death.time">run time</small></div>
                <div class="wb-rs"><span id="wbRecapBest">0</span><small data-wb="death.best">best</small></div>
            </div>
            <div class="wb-killer" style="display:none;"></div>'''
assert old in s; s = s.replace(old, new, 1)

anchor = '</head>'
css = '''
    <style>
        /* ---------- RICHNESS Н5: кинематограф ---------- */
        .wb-mk {
            font-family: var(--wb-font); font-size: 44px; font-weight: 700;
            letter-spacing: 8px; color: #fff; text-transform: uppercase;
            text-shadow: 0 0 16px rgba(0,240,255,0.8), 3px 0 0 rgba(255,46,136,0.5), -3px 0 0 rgba(0,240,255,0.5);
            animation: wbBannerIn .3s cubic-bezier(.2,.9,.3,1.3);
        }
        #wbMultikill, #wbBossIntro, #wbZoneTitle {
            position: fixed; inset: 0; z-index: 240; pointer-events: none;
            display: none; align-items: center; justify-content: center; flex-direction: column;
        }
        .wb-bi-label {
            font-family: var(--wb-mono); font-size: 15px; letter-spacing: 8px;
            color: var(--wb-magenta); text-transform: uppercase;
        }
        .wb-bi-name {
            font-family: var(--wb-font); font-size: clamp(40px, 9vh, 84px);
            font-weight: 700; letter-spacing: 4px; margin: 8px 0;
            animation: wbBannerIn .4s cubic-bezier(.2,.9,.3,1.2);
        }
        .wb-bi-sub {
            font-family: var(--wb-mono); font-size: 13px; letter-spacing: 5px;
            color: var(--wb-dim); text-transform: uppercase;
        }
        .wb-zt {
            font-family: var(--wb-font); font-size: clamp(52px, 11vh, 100px);
            font-weight: 700; letter-spacing: 12px; text-transform: uppercase;
            color: #fff; text-shadow: 0 0 30px rgba(255,255,255,0.4), 0 6px 40px rgba(0,0,0,0.6);
            animation: wbBannerIn .5s cubic-bezier(.2,.9,.3,1.15);
        }
        .wb-killer {
            display: flex; align-items: center; gap: 10px; margin-top: 4px;
        }
        .wb-killer-by {
            font-family: var(--wb-mono); font-size: 11px; letter-spacing: 2px;
            color: var(--wb-dim); text-transform: uppercase;
        }
        .wb-killer b {
            font-family: var(--wb-font); font-size: 15px; font-weight: 600;
            letter-spacing: 1px; text-transform: uppercase;
        }
        .wb-killer-rar {
            font-family: var(--wb-mono); font-size: 10.5px; letter-spacing: 1px;
            border: 1px solid; border-radius: 6px; padding: 2px 8px;
        }
        /* Экран загрузки WAVEBORN: лого + советы */
        #loading { flex-direction: column; }
        #loading .wb-load-logo {
            font-family: var(--wb-font); font-size: 34px; font-weight: 700;
            letter-spacing: 10px; color: #fff; text-transform: uppercase;
            text-shadow: 0 0 14px rgba(0,240,255,0.8), 2px 0 0 rgba(255,46,136,0.5);
            margin-bottom: 18px;
        }
        #loading .wb-load-tip {
            font-family: var(--wb-mono); font-size: 13px; letter-spacing: 1px;
            color: var(--wb-dim); max-width: 560px; text-align: center; margin-top: 14px;
            min-height: 18px;
        }
        #loading .wb-load-bar {
            width: 300px; height: 4px; background: rgba(255,255,255,0.10);
            border-radius: 3px; overflow: hidden;
        }
        #loading .wb-load-bar i {
            display: block; height: 100%; width: 40%;
            background: linear-gradient(90deg, var(--wb-cyan), var(--wb-lime));
            animation: wbLoad 1.1s ease-in-out infinite;
        }
        @keyframes wbLoad {
            0% { transform: translateX(-100%); }
            100% { transform: translateX(320%); }
        }
    </style>
'''
i = s.find(anchor)
s = s[:i] + css + '\n' + s[i:]
io.open(p, 'w', encoding='utf-8', newline='').write(s)
print('index.html OK')

# ===== 5. Загрузка: лого + советы (main.js showLoading) =====
p = 'client/js/main.js'
s = io.open(p, encoding='utf-8').read()
old = '''function showLoading(text) {
    try { if (typeof stopWaveDecor === 'function') stopWaveDecor(); } catch (e) {}'''
new = '''const WB_TIPS = [
    'Every 5th wave — Swarm / Tanks / Hunters. Every 10th — BOSS.',
    'Golden mobs drop x10 loot. Ring/Star/Crescent boost drop rarity.',
    'Craft 5 same shapes → next rarity. Grades: UNIQUE → PARAGON.',
    'Absorb spare petals for XP — the price is capped, no exploit.',
    'Hexagon+ crafts have a damage floor — no weak crafts.',
    'Click a mob card in the roster to highlight it on the arena.',
];
let wbTipTimer = 0, wbTipIdx = 0;
function showLoading(text) {
    try { if (typeof stopWaveDecor === 'function') stopWaveDecor(); } catch (e) {}
    // WAVEBORN: лого + советы на экране загрузки.
    try {
        const el = document.getElementById('loading');
        if (el && !el.querySelector('.wb-load-logo')) {
            const logo = document.createElement('div');
            logo.className = 'wb-load-logo';
            logo.textContent = 'WAVEBORN';
            const bar = document.createElement('div');
            bar.className = 'wb-load-bar';
            bar.innerHTML = '<i></i>';
            const tip = document.createElement('div');
            tip.className = 'wb-load-tip';
            el.appendChild(logo);
            el.appendChild(bar);
            el.appendChild(tip);
            wbTipIdx = Math.floor(Math.random() * WB_TIPS.length);
            tip.textContent = WB_TIPS[wbTipIdx % WB_TIPS.length];
            wbTipTimer = setInterval(() => {
                wbTipIdx++;
                tip.textContent = WB_TIPS[wbTipIdx % WB_TIPS.length];
            }, 2600);
        }
        const lt = document.getElementById('loadingText');
        if (lt) lt.style.display = 'none';
    } catch (e) {}'''
assert old in s, 'showLoading'
s = s.replace(old, new, 1)

# Скрытие загрузки — убрать таймер советов. Найти hideLoading.
old = '''function hideLoading() {'''
if old in s:
    new = '''function hideLoading() {
    try { if (wbTipTimer) { clearInterval(wbTipTimer); wbTipTimer = 0; } } catch (e) {}'''
    s = s.replace(old, new, 1)
io.open(p, 'w', encoding='utf-8', newline='').write(s)
print('loading OK')

# ===== 6. Музыка: mood в showScreen =====
old = '''    // Фон WAVEBORN: волны-осциллографы на экранах меню, стоп в игре.'''
p = 'client/js/main.js'
s = io.open(p, encoding='utf-8').read()
new = '''    // WAVEBORN: процедурная музыка меню (audio.js mood-система).
    try {
        if (window.GameAudio) {
            if (name === 'menu' || name === 'login' || name === 'register') window.GameAudio.mood('menu');
            else window.GameAudio.mood('off');
        }
    } catch (e) {}
    // Фон WAVEBORN: волны-осциллографы на экранах меню, стоп в игре.'''
assert old in s; s = s.replace(old, new, 1)
io.open(p, 'w', encoding='utf-8', newline='').write(s)
print('music OK')
