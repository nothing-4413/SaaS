const $ = (selector) => document.querySelector(selector);
const $$ = (selector) => [...document.querySelectorAll(selector)];
const state = { token: localStorage.getItem('sp_token') || '', refreshToken: localStorage.getItem('sp_refresh_token') || '', org: localStorage.getItem('sp_org') || '', products: [], skus: [], warehouses: [] };

function message(text = '', error = false) { const el = $('#app-message'); if (el) { el.textContent = text; el.style.color = error ? '#b93f39' : '#087f72'; } }
function authMessage(text = '', error = true) { const el = $('#auth-message'); el.textContent = text; el.style.color = error ? '#b93f39' : '#087f72'; }
function money(cents = 0) { return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(cents / 100); }
function escapeHtml(value = '') { return String(value).replace(/[&<>'"]/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;',"'":'&#39;','"':'&quot;'}[c])); }
function id(value = '') { return value.slice(0, 8); }
function nowKey(prefix) { return `${prefix}-${Date.now()}`; }

async function api(path, options = {}) {
  const headers = { ...(options.body ? {'Content-Type':'application/json'} : {}), ...(options.headers || {}) };
  if (state.token) headers.Authorization = `Bearer ${state.token}`;
  const response = await fetch(path, { ...options, headers });
  if (response.status === 401 && state.refreshToken && !options.retried && state.org) {
    const refreshed = await fetch(orgPath('/sessions'), {method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify({refresh_token:state.refreshToken})});
    if (refreshed.ok) { state.token = (await refreshed.json()).access_token; localStorage.setItem('sp_token', state.token); return api(path, {...options, retried:true}); }
  }
  if (response.status === 204) return null;
  const data = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(data.error || `Request failed (${response.status})`);
  return data;
}
function orgPath(suffix) { return `/organizations/${encodeURIComponent(state.org)}${suffix}`; }
function listTable(headers, rows) {
  if (!rows.length) return '<p class="empty">No records yet.</p>';
  return `<table class="data-table"><thead><tr>${headers.map(h => `<th>${h}</th>`).join('')}</tr></thead><tbody>${rows.join('')}</tbody></table>`;
}
function optionMarkup(items, label) { return `<option value="">Select...</option>${items.map(v => `<option value="${v.id}">${escapeHtml(label(v))}</option>`).join('')}`; }

function setAuthMode(mode) {
  const reset = mode === 'reset', setup = mode === 'setup', forgot = mode === 'forgot';
  $('#login-form').classList.toggle('hidden', setup || reset || forgot); $('#setup-form').classList.toggle('hidden', !setup); $('#reset-form').classList.toggle('hidden', !reset); $('#forgot-form').classList.toggle('hidden', !forgot);
  $('#switch-auth').classList.toggle('hidden', reset); $('#forgot-password').classList.toggle('hidden', reset || forgot || setup);
  $('#auth-title').textContent = reset ? 'Set a new password' : forgot ? 'Reset password' : setup ? 'Create workspace' : 'Sign in';
  $('#auth-help').textContent = reset ? 'Choose a new password for your workspace account.' : forgot ? 'A secure reset link will be sent to your email.' : setup ? 'Start with an owner account and full workspace access.' : 'Secure access for inventory teams.';
  $('#switch-auth').textContent = setup || forgot ? 'Back to sign in' : 'Create a new workspace';
}
async function enterApp() {
  $('#auth-view').classList.add('hidden'); $('#app-view').classList.remove('hidden'); $('#org-label').textContent = state.org;
  await refresh();
}
function storeSession(token, refreshToken, org) { state.token = token; state.refreshToken = refreshToken; state.org = org; localStorage.setItem('sp_token', token); localStorage.setItem('sp_refresh_token', refreshToken); localStorage.setItem('sp_org', org); }
function clearSession() { state.token = ''; state.refreshToken = ''; state.org = ''; localStorage.removeItem('sp_token'); localStorage.removeItem('sp_refresh_token'); localStorage.removeItem('sp_org'); $('#app-view').classList.add('hidden'); $('#auth-view').classList.remove('hidden'); setAuthMode('login'); }

async function refresh() {
  if (!state.org) return;
  message('');
  try {
    const [orders, stocks, products, warehouses, rule] = await Promise.all([
      api(orgPath('/orders?page_size=100&order=desc')), api(orgPath('/stocks?page_size=100&sort=available')), api(orgPath('/products?page_size=100&sort=name')), api(orgPath('/warehouses?page_size=100&sort=name')), api(orgPath('/alerts/stock')).catch(() => null)
    ]);
    const summary = await api(orgPath(`/reports/summary?low_stock_threshold=${rule && rule.enabled ? rule.threshold : 0}`));
    state.products = products.items || []; state.warehouses = warehouses.items || [];
    const skuPages = await Promise.all(state.products.map(product => api(orgPath(`/products/${product.id}/skus?page_size=100`))));
    state.skus = skuPages.flatMap(page => page.items || []);
    render(summary, orders.items || [], stocks.items || [], rule);
    $('#sync-time').textContent = `Synced ${new Date().toLocaleTimeString([], {hour:'2-digit', minute:'2-digit'})}`;
  } catch (error) { message(error.message, true); if (/401|token|session/i.test(error.message)) clearSession(); }
}
function render(summary, orders, stocks, rule) {
  const inv = summary.inventory || {}, os = summary.orders || {};
  $('#metric-revenue').textContent = money(os.confirmed_cents); $('#metric-open-orders').textContent = (os.pending||0)+(os.confirmed||0)+(os.paid||0)+(os.shipped||0); $('#metric-available').textContent = inv.available || 0; $('#metric-low-stock').textContent = inv.low_stock || 0;
  $('#alert-threshold').textContent = `Threshold: ${rule ? rule.threshold : 0} units`; $('#alert-form [name=threshold]').value = rule ? rule.threshold : 0; $('#alert-form [name=enabled]').checked = Boolean(rule && rule.enabled);
  $('#order-flow').innerHTML = [['Pending',os.pending],['Confirmed',os.confirmed],['Paid',os.paid],['Shipping',os.shipped]].map(([name,count]) => `<div><strong>${count || 0}</strong><span>${name}</span></div>`).join('');
  $('#recent-orders').innerHTML = ordersTable(orders.slice(0, 6), false); $('#orders-table').innerHTML = ordersTable(orders, true); $('#order-count').textContent = `${orders.length} total`;
  $('#stock-table').innerHTML = stocksTable(stocks); $('#catalog-table').innerHTML = catalogTable(); fillSelects();
}
function ordersTable(orders, actions) { return listTable(['Order','Status','Total','Created', actions ? 'Action' : ''], orders.map(o => `<tr><td title="${o.id}">#${id(o.id)}</td><td><span class="status ${o.status}">${o.status}</span></td><td>${money(o.total_cents)}</td><td>${new Date(o.created_at).toLocaleDateString()}</td><td>${actions ? orderActions(o) : ''}</td></tr>`)); }
function orderActions(order) { const next = {pending:'confirm',confirmed:'pay',paid:'ship',shipped:'complete'}[order.status]; const buttons = next ? `<button class="small-action" data-order-action="${next}" data-order-id="${order.id}">${next}</button>` : ''; return buttons + (order.status === 'pending' ? `<button class="small-action" data-order-action="cancel" data-order-id="${order.id}">cancel</button>` : '') + (['paid','shipped','completed'].includes(order.status) ? `<button class="small-action" data-order-action="refund" data-order-id="${order.id}">refund</button>` : ''); }
function stocksTable(stocks) { return listTable(['Warehouse','SKU','On hand','Reserved','Available','Updated'], stocks.map(s => `<tr><td title="${s.warehouse_id}">${id(s.warehouse_id)}</td><td title="${s.sku_id}">${skuName(s.sku_id)}</td><td>${s.on_hand}</td><td>${s.reserved}</td><td>${s.available}</td><td>${new Date(s.updated_at).toLocaleDateString()}</td></tr>`)); }
function catalogTable() { return listTable(['Product','SKU code','SKU name','Price'], state.skus.map(s => `<tr><td>${escapeHtml(productName(s.product_id))}</td><td>${escapeHtml(s.code)}</td><td>${escapeHtml(s.name)}</td><td>${money(s.price_cents)}</td></tr>`)); }
function skuName(skuID) { const sku = state.skus.find(s => s.id === skuID); return sku ? escapeHtml(`${sku.code} · ${sku.name}`) : id(skuID); }
function productName(productID) { return (state.products.find(p => p.id === productID) || {}).name || id(productID); }
function fillSelects() {
  $$('select[name=warehouse_id]').forEach(select => { const selected = select.value; select.innerHTML = optionMarkup(state.warehouses, w => w.name); select.value = selected; });
  $$('select[name=sku_id]').forEach(select => { const selected = select.value; select.innerHTML = optionMarkup(state.skus, s => `${s.code} · ${s.name}`); select.value = selected; });
  const productSelect = $('#sku-form [name=product_id]'); if (productSelect) productSelect.innerHTML = optionMarkup(state.products, p => p.name);
}

$('#login-form').addEventListener('submit', async event => { event.preventDefault(); const data = Object.fromEntries(new FormData(event.target)); authMessage(''); try { const result = await api(`/organizations/${encodeURIComponent(data.organization_id)}/sessions`, {method:'POST',body:JSON.stringify({email:data.email,password:data.password})}); storeSession(result.access_token, result.refresh_token, data.organization_id); await enterApp(); } catch (error) { authMessage(error.message); } });
$('#setup-form').addEventListener('submit', async event => { event.preventDefault(); const data = Object.fromEntries(new FormData(event.target)); authMessage(''); try { const org = await api('/organizations', {method:'POST',body:JSON.stringify(data)}); authMessage(`Workspace created. Your organization ID is ${org.id}. Sign in with it.`, false); $('#login-form [name=organization_id]').value = org.id; $('#login-form [name=email]').value = data.owner_email; setAuthMode('login'); } catch (error) { authMessage(error.message); } });
$('#reset-form').addEventListener('submit', async event => { event.preventDefault(); const data = Object.fromEntries(new FormData(event.target)), params = new URLSearchParams(location.search); if (data.password !== data.confirm_password) return authMessage('Passwords do not match.'); try { await api(`/organizations/${encodeURIComponent(params.get('organization_id') || '')}/sessions/password-reset`, {method:'PUT',body:JSON.stringify({token:params.get('token'),password:data.password})}); authMessage('Password updated. You can now sign in.', false); history.replaceState({}, '', '/'); setAuthMode('login'); } catch (error) { authMessage(error.message); } });
$('#forgot-form').addEventListener('submit', async event => { event.preventDefault(); const data = Object.fromEntries(new FormData(event.target)); try { await api(`/organizations/${encodeURIComponent(data.organization_id)}/sessions/password-reset`, {method:'POST',body:JSON.stringify({email:data.email})}); authMessage('If the account exists, a reset email is on its way.', false); } catch (error) { authMessage(error.message); } });
$('#switch-auth').addEventListener('click', () => setAuthMode($('#switch-auth').textContent.includes('Back') ? 'login' : 'setup'));
$('#forgot-password').addEventListener('click', () => setAuthMode('forgot'));
$('#logout').addEventListener('click', clearSession); $('#refresh').addEventListener('click', refresh);
$$('.nav-item').forEach(button => button.addEventListener('click', () => showView(button.dataset.view))); $$('[data-view-target]').forEach(button => button.addEventListener('click', () => showView(button.dataset.viewTarget)));
function showView(view) { $$('.view').forEach(el => el.classList.toggle('hidden', el.id !== view)); $$('.nav-item').forEach(el => el.classList.toggle('active', el.dataset.view === view)); $('#view-title').textContent = ({overview:'Business overview',inventory:'Inventory control',orders:'Order fulfillment',catalog:'Catalog setup'})[view]; $('#view-kicker').textContent = ({overview:'LIVE OPERATIONS',inventory:'STOCK LEDGER',orders:'FULFILLMENT QUEUE',catalog:'MASTER DATA'})[view]; }
$('#alert-form').addEventListener('submit', async event => { event.preventDefault(); const form = new FormData(event.target); try { await api(orgPath('/alerts/stock'), {method:'PUT',body:JSON.stringify({threshold:Number(form.get('threshold')),enabled:form.get('enabled') === 'on'})}); message('Low-stock policy saved.'); refresh(); } catch (error) { message(error.message,true); } });
$('#receive-form').addEventListener('submit', async event => { event.preventDefault(); const data = Object.fromEntries(new FormData(event.target)); try { await api(orgPath(`/warehouses/${data.warehouse_id}/skus/${data.sku_id}/stock/receive`), {method:'POST',body:JSON.stringify({quantity:Number(data.quantity),idempotency_key:data.idempotency_key})}); message('Inventory received.'); event.target.reset(); refresh(); } catch (error) { message(error.message,true); } });
$('#order-form').addEventListener('submit', async event => { event.preventDefault(); const data = Object.fromEntries(new FormData(event.target)); try { await api(orgPath('/orders'), {method:'POST',body:JSON.stringify({idempotency_key:data.idempotency_key,lines:[{warehouse_id:data.warehouse_id,sku_id:data.sku_id,quantity:Number(data.quantity)}]})}); message('Order created and stock reserved.'); event.target.reset(); refresh(); } catch (error) { message(error.message,true); } });
$('#orders-table').addEventListener('click', async event => { const button = event.target.closest('[data-order-action]'); if (!button) return; try { await api(orgPath(`/orders/${button.dataset.orderId}/${button.dataset.orderAction}`), {method:'POST'}); message(`Order ${button.dataset.orderAction}ed.`); refresh(); } catch (error) { message(error.message,true); } });
$('#warehouse-form').addEventListener('submit', async event => { event.preventDefault(); const data = Object.fromEntries(new FormData(event.target)); try { await api(orgPath('/warehouses'), {method:'POST',body:JSON.stringify(data)}); message('Warehouse added.'); event.target.reset(); refresh(); } catch (error) { message(error.message,true); } });
$('#product-form').addEventListener('submit', async event => { event.preventDefault(); const data = Object.fromEntries(new FormData(event.target)); try { await api(orgPath('/products'), {method:'POST',body:JSON.stringify(data)}); message('Product added. Add its SKU next.'); event.target.reset(); refresh(); } catch (error) { message(error.message,true); } });
$('#open-sku-form').addEventListener('click', () => $('#sku-surface').classList.remove('hidden')); $('#close-sku-form').addEventListener('click', () => $('#sku-surface').classList.add('hidden'));
$('#sku-form').addEventListener('submit', async event => { event.preventDefault(); const data = Object.fromEntries(new FormData(event.target)); try { await api(orgPath(`/products/${data.product_id}/skus`), {method:'POST',body:JSON.stringify({...data,price_cents:Number(data.price_cents)})}); message('SKU added.'); event.target.reset(); $('#sku-surface').classList.add('hidden'); refresh(); } catch (error) { message(error.message,true); } });
$('#download-stock').addEventListener('click', async () => { try { const response = await fetch(orgPath('/exports/stocks/csv'), {headers:{Authorization:`Bearer ${state.token}`}}); if (!response.ok) throw new Error('CSV export failed'); const url = URL.createObjectURL(await response.blob()), link = Object.assign(document.createElement('a'), {href:url,download:'stocks.csv'}); link.click(); URL.revokeObjectURL(url); } catch (error) { message(error.message,true); } });

if (location.pathname === '/reset-password') setAuthMode('reset'); else if (state.token && state.org) enterApp();
