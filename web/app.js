'use strict';

const token = document.querySelector('meta[name="token"]').content;
const $ = (id) => document.getElementById(id);

// All text goes through textContent: case folder names come from other people's uploads.
function el(tag, cls, text) {
  const e = document.createElement(tag);
  if (cls) e.className = cls;
  if (text !== undefined) e.textContent = text;
  return e;
}

async function api(method, path) {
  const r = await fetch(path, {
    method,
    headers: { 'X-Token': token, 'Content-Type': 'application/json' },
    body: method === 'GET' ? undefined : '{}',
  });
  const j = await r.json().catch(() => ({}));
  if (!r.ok) throw new Error(j.error || ('HTTP ' + r.status));
  return j;
}

function showError(e) {
  const b = $('error');
  b.hidden = !e;
  b.textContent = e ? String(e.message || e) : '';
}

function tile(cls, n, label) {
  const t = el('div', 'tile ' + cls);
  t.append(el('div', 'n', String(n)), el('div', 'l', label));
  return t;
}

function renderState(s) {
  const st = s.stats;
  $('root').textContent = s.root + '  ·  v' + s.version;

  const tiles = $('tiles');
  tiles.replaceChildren(
    tile('', st.total, '已分類總計'),
    tile('ios', st.ios, 'IOS'),
    tile('die', st.die, 'Die'),
    tile('', st.pending, '待處理'),
  );

  const tb = $('days').tBodies[0];
  tb.replaceChildren();
  for (const d of st.days) {
    const tr = el('tr');
    tr.append(el('td', '', d.date), el('td', 'num', String(d.ios)), el('td', 'num', String(d.die)), el('td', 'num', String(d.ios + d.die)));
    tb.append(tr);
  }
  $('daysEmpty').hidden = st.days.length > 0;

  const ul = $('pendingList');
  ul.replaceChildren();
  for (const b of st.pendingBatches) {
    ul.append(el('li', '', b.date + '　' + b.name + '　（' + b.cases + ' 個 case）'));
  }
  $('pendingEmpty').hidden = st.pendingBatches.length > 0;
  $('previewBtn').disabled = st.pendingBatches.length === 0;
}

function renderTeeth(t) {
  const tb = $('teeth').tBodies[0];
  tb.replaceChildren();
  for (const x of t.teeth) {
    const tr = el('tr');
    tr.append(el('td', '', x.fdi), el('td', 'num', String(x.ios)), el('td', 'num', String(x.die)), el('td', 'num', String(x.ios + x.die)));
    tb.append(tr);
  }
  const tf = $('teeth').tFoot;
  tf.replaceChildren();
  const tr = el('tr', 'total');
  tr.append(el('td', '', '總量'), el('td', 'num', String(t.iosTeeth)), el('td', 'num', String(t.dieTeeth)), el('td', 'num', String(t.iosTeeth + t.dieTeeth)));
  tf.append(tr);
  $('teeth').hidden = t.teeth.length === 0;
  $('teethNote').textContent = t.teeth.length
    ? '每個 case 的每個牙位算一次（IOS 看 …-XX-crown_cad.stl，Die 看 …-Tooth_XX.stl）；共 ' +
      (t.iosCases + t.dieCases) + ' 個 case。'
    : '還沒有整理好的資料。';
  const nf = $('noFdi');
  nf.hidden = t.noFdi === 0;
  nf.textContent = t.noFdi
    ? '有 ' + t.noFdi + ' 個 case 找不到牙位檔，未計入：' + t.noFdiList.join('、') + (t.noFdi > t.noFdiList.length ? '…' : '')
    : '';
}

async function refreshTeeth() {
  $('teethNote').textContent = '統計中…';
  try {
    renderTeeth(await api('GET', '/api/teeth'));
  } catch (e) {
    $('teethNote').textContent = '牙位統計失敗：' + String(e.message || e);
  }
}

async function refresh() {
  try {
    showError(null);
    renderState(await api('GET', '/api/state'));
  } catch (e) {
    showError(e);
  }
  refreshTeeth();
}

function note(it) {
  if (it.conflict) return it.conflict + '（將略過）';
  const parts = [];
  if (it.basis) parts.push(it.basis);
  if (it.flatten > 0) parts.push('展平 ' + it.flatten + ' 層重複資料夾');
  return parts.join('；');
}

function renderPlan(p) {
  const movable = p.items.filter((i) => !i.conflict);
  const ios = movable.filter((i) => i.kind === 'IOS').length;
  const die = movable.filter((i) => i.kind === 'Die').length;
  const skipped = p.items.length - movable.length;
  $('planSummary').textContent =
    '共 ' + p.items.length + ' 個 case：移到 IOS ' + ios + ' 個、移到 Die ' + die + ' 個' +
    (skipped ? '、略過 ' + skipped + ' 個' : '') + '。';

  const tb = $('planTable').tBodies[0];
  tb.replaceChildren();
  for (const it of p.items) {
    const tr = el('tr');
    const badge = el('span', 'badge ' + (it.conflict ? 'skip' : it.kind), it.conflict ? '略過' : it.kind);
    const kind = el('td');
    kind.append(badge);
    tr.append(el('td', '', it.batch), el('td', '', it.case), kind, el('td', 'muted', note(it)));
    tb.append(tr);
  }
  const ig = $('planIgnored');
  ig.hidden = p.ignored.length === 0;
  ig.textContent = p.ignored.length ? '批次資料夾內的非 case 項目，不會動：' + p.ignored.join('、') : '';
  $('applyBtn').disabled = movable.length === 0;
  $('planSection').hidden = false;
  $('resultSection').hidden = true;
}

function renderResult(r) {
  $('resultSummary').textContent =
    '已移動 ' + r.moved + ' 個 case' + (r.failed ? '，失敗 ' + r.failed + ' 個' : '') +
    (r.log ? '。紀錄檔：' + r.log : '。');
  const ul = $('resultErrors');
  ul.replaceChildren();
  for (const e of r.errors) ul.append(el('li', '', e));
  $('resultSection').hidden = false;
  $('planSection').hidden = true;
}

$('refresh').addEventListener('click', refresh);

$('previewBtn').addEventListener('click', async () => {
  try {
    showError(null);
    renderPlan(await api('POST', '/api/plan'));
  } catch (e) {
    showError(e);
  }
});

$('cancelBtn').addEventListener('click', () => {
  $('planSection').hidden = true;
});

$('applyBtn').addEventListener('click', async () => {
  const b = $('applyBtn');
  b.disabled = true;
  try {
    showError(null);
    renderResult(await api('POST', '/api/apply'));
    await refresh();
  } catch (e) {
    showError(e);
    b.disabled = false;
  }
});

refresh();
