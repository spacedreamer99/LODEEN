const canvas = document.getElementById('c');
const ctx = canvas.getContext('2d');
const statusEl = document.getElementById('status');
const tickEl = document.getElementById('tick');
const playersEl = document.getElementById('players');
const logEl = document.getElementById('log');

let world = { players: [], tick: 0 };
let myId = null;

function resize() {
  canvas.width = window.innerWidth;
  canvas.height = window.innerHeight;
}
window.addEventListener('resize', resize);
resize();

function log(msg) {
  const line = document.createElement('div');
  line.textContent = `[${new Date().toLocaleTimeString()}] ${msg}`;
  logEl.appendChild(line);
  logEl.scrollTop = logEl.scrollHeight;
}

const ws = new WebSocket(`ws://${location.host}/ws`);

ws.onopen = () => {
  statusEl.textContent = 'connected';
  log('WS open');
  send('hello', { nick: 'web-' + Math.floor(Math.random()*1000), version: 'web', color_hex: '#4af' });
};

ws.onclose = () => { statusEl.textContent = 'disconnected'; log('WS close'); };
ws.onerror = (e) => log('WS error: ' + (e.message || ''));

ws.onmessage = (ev) => {
  let env;
  try { env = JSON.parse(ev.data); } catch { return; }
  handle(env);
};

function send(type, data) {
  if (ws.readyState !== WebSocket.OPEN) return;
  ws.send(JSON.stringify({ type, data }));
}

function handle(env) {
  switch (env.type) {
    case 'welcome': {
      const w = env.data || {};
      myId = w.player_id || w.PlayerID;
      log(`welcome: player_id=${myId}`);
      break;
    }
    case 'snapshot': {
      world = env.data || {};
      tickEl.textContent = world.tick ?? '—';
      playersEl.textContent = (world.players || []).length;
      break;
    }
    case 'chat': {
      const m = env.data || {};
      log(`chat ${m.from}: ${m.text}`);
      break;
    }
    case 'pong':
      break;
    default:
      break;
  }
}

function draw() {
  ctx.fillStyle = '#111';
  ctx.fillRect(0, 0, canvas.width, canvas.height);

  const players = world.players || [];
  const cx = canvas.width / 2;
  const cy = canvas.height / 2;
  const scale = 2;

  for (const p of players) {
    const x = cx + (p.x ?? 0) * scale;
    const y = cy - (p.y ?? 0) * scale;
    ctx.beginPath();
    ctx.arc(x, y, p.id === myId ? 8 : 6, 0, Math.PI * 2);
    ctx.fillStyle = p.id === myId ? '#4af' : '#f84';
    ctx.fill();
    ctx.fillStyle = '#fff';
    ctx.fillText(p.nick || p.id, x + 10, y + 4);
  }

  if (world.earth_pos || world.EarthPos) {
    const e = world.earth_pos || world.EarthPos;
    ctx.fillStyle = '#2a2';
    ctx.fillText(`earth: ${e.x?.toFixed(1)},${e.y?.toFixed(1)},${e.z?.toFixed(1)}`, 20, 80);
  }

  requestAnimationFrame(draw);
}
draw();

setInterval(() => send('ping', { sent: Date.now() }), 1000);
