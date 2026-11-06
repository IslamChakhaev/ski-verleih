"use strict";

// The frontend only displays data and forwards input to the Go API. All
// business rules and price calculations live in the backend.

const SUCCESS_MESSAGE_MS = 5000;

const euroFormat = new Intl.NumberFormat("de-DE", { style: "currency", currency: "EUR" });
const dateFormat = new Intl.DateTimeFormat("de-DE", { dateStyle: "medium", timeStyle: "short" });

const byId = (id) => document.getElementById(id);
const formatEuro = (cents) => euroFormat.format(cents / 100);
const formatDate = (value) => dateFormat.format(new Date(value));

// Works for rentals and price quotes, which share these fields.
const formatPrice = ({ durationHours, hourlyPriceCents, totalPriceCents }) =>
  `${durationHours} Std. × ${formatEuro(hourlyPriceCents)} = ${formatEuro(totalPriceCents)}`;

// ---------- API ----------

// api returns the parsed response or throws an Error with a German message.
// Server errors are JSON, but Go's router answers unknown routes in plain
// English text, which is logged and replaced by a German message.
async function api(path, options = {}) {
  let response;
  try {
    response = await fetch(path, options);
  } catch {
    throw new Error("Der Server ist nicht erreichbar. Bitte prüfen Sie, ob die Anwendung läuft.");
  }

  const text = await response.text();
  let data = null;
  try {
    data = JSON.parse(text);
  } catch {
    console.warn(`Keine JSON-Antwort von ${path}:`, text);
  }

  if (!response.ok) {
    throw new Error(data?.error ?? `Die Anfrage ist fehlgeschlagen (Status ${response.status}).`);
  }
  if (data === null) {
    throw new Error("Der Server hat eine unerwartete Antwort gesendet.");
  }
  return data;
}

function sendJSON(path, method, body) {
  return api(path, {
    method,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}

// ---------- Feedback ----------

let messageTimer;

function hideMessage() {
  clearTimeout(messageTimer);
  byId("message").hidden = true;
}

function showMessage(text, type) {
  hideMessage();
  const box = byId("message");
  box.textContent = text;
  box.className = `message message-${type}`;
  box.hidden = false;
  if (type === "success") {
    messageTimer = setTimeout(hideMessage, SUCCESS_MESSAGE_MS);
  }
}

// ---------- DOM helpers ----------
// User data only reaches the page as text (textContent, append, new Option),
// so it can never be interpreted as HTML.

function createElement(tag, className, text) {
  const node = document.createElement(tag);
  node.className = className;
  node.textContent = text;
  return node;
}

const badge = (text, kind) => createElement("span", `badge badge-${kind}`, text);
const noWrap = (text) => createElement("span", "nowrap", text);

function actionButton(text, onClick) {
  const button = createElement("button", "button button-small", text);
  button.type = "button";
  button.addEventListener("click", () => onClick(button));
  return button;
}

function showTableNote(tbody, text) {
  const cell = createElement("td", "table-note", text);
  cell.colSpan = tbody.closest("table").tHead.rows[0].cells.length;
  const row = document.createElement("tr");
  row.append(cell);
  tbody.replaceChildren(row);
}

// renderRows fills a table. Each row is a list of strings or DOM nodes. Cells
// carry their column name, which the CSS shows as a label on small screens.
function renderRows(tbody, rows, emptyText) {
  if (rows.length === 0) {
    showTableNote(tbody, emptyText);
    return;
  }
  const headers = [...tbody.closest("table").tHead.rows[0].cells];
  tbody.replaceChildren(...rows.map((cells) => {
    const row = document.createElement("tr");
    cells.forEach((content, index) => {
      const cell = document.createElement("td");
      cell.dataset.label = headers[index].textContent;
      cell.append(content);
      row.append(cell);
    });
    return row;
  }));
}

// fillSelect replaces the options and keeps the selection if it still exists.
function fillSelect(select, items, toLabel, emptyLabel) {
  const selected = select.value;
  const options = items.map((item) => new Option(toLabel(item), item.id));
  const placeholder = new Option(items.length > 0 ? "Bitte wählen …" : emptyLabel, "");
  select.replaceChildren(placeholder, ...options);
  select.value = options.some((option) => option.value === selected) ? selected : "";
}

// ---------- Forms ----------

// showFormError shows an error inside the dialog, because the page-level
// message would be hidden behind an open dialog.
function showFormError(form, text) {
  const box = form.querySelector(".form-error");
  box.textContent = text;
  box.hidden = text === "";
  if (text) {
    box.scrollIntoView({ block: "nearest" });
  }
}

function openDialog(form) {
  hideMessage();
  showFormError(form, "");
  const dialog = form.closest("dialog");
  if (!dialog.open) {
    dialog.showModal();
  }
}

// restoreFocus moves focus back to the page if a list refresh removed the
// focused button. It does not scroll, so mouse users stay where they are.
function restoreFocus() {
  if (document.activeElement === document.body) {
    document.querySelector(".view:not([hidden]) .page-header .button").focus({ preventScroll: true });
  }
}

async function finishDialog(form, message) {
  form.closest("dialog").close();
  showMessage(message, "success");
  await refreshView();
  restoreFocus();
}

// handleSubmit checks that the form is complete, runs action and reports
// errors. It only checks the input format; the backend makes the final decision.
function handleSubmit(form, action) {
  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    showFormError(form, "");

    const invalidField = [...form.elements].find((field) => !field.checkValidity());
    if (invalidField) {
      showFormError(form, invalidField.dataset.error);
      invalidField.focus();
      return;
    }

    const submitButton = form.querySelector("[type=submit]");
    submitButton.disabled = true;
    try {
      await action(Object.fromEntries(new FormData(form)));
    } catch (error) {
      showFormError(form, error.message);
    } finally {
      submitButton.disabled = false;
    }
  });
}

// createEditableForm connects a dialog form that both creates and edits an item.
// Calling open() starts a new item, and open(item) edits that item. toBody turns
// form values into the request body; toFormValues does the opposite. A caller
// that passes onCreated to open() decides what happens after creating the item.
function createEditableForm({ formId, path, noun, toBody = (values) => values, toFormValues = (item) => item }) {
  const form = byId(formId);
  const title = form.querySelector("h2");
  const submitButton = form.querySelector("[type=submit]");
  const createTitle = title.textContent;
  const createLabel = submitButton.textContent;
  let editingId = null;
  let afterCreate = null;

  function open(item = null, onCreated = null) {
    editingId = item?.id ?? null;
    afterCreate = onCreated;
    form.reset();
    if (item) {
      const values = toFormValues(item);
      for (const field of form.elements) {
        if (field.name) {
          field.value = values[field.name];
        }
      }
    }
    form.classList.toggle("editing", item !== null);
    title.textContent = item ? `${noun} bearbeiten` : createTitle;
    submitButton.textContent = item ? "Änderungen speichern" : createLabel;
    openDialog(form);
  }

  handleSubmit(form, async (values) => {
    const body = toBody(values);
    if (editingId === null) {
      const created = await sendJSON(path, "POST", body);
      if (afterCreate) {
        await afterCreate(created);
      } else {
        await finishDialog(form, `${noun} angelegt.`);
      }
    } else {
      await sendJSON(`${path}/${editingId}`, "PUT", body);
      await finishDialog(form, "Änderungen gespeichert.");
    }
  });

  return { open };
}

const skiForm = createEditableForm({ formId: "ski-form", path: "/api/skis", noun: "Ski" });

const customerForm = createEditableForm({ formId: "customer-form", path: "/api/customers", noun: "Person" });

const tariffForm = createEditableForm({
  formId: "tariff-form",
  path: "/api/tariffs",
  noun: "Tarif",
  // The price is a text field because number fields treat "6,50" differently
  // depending on the browser language (as 6.50 or as 650).
  toBody({ name, price }) {
    const hourlyPriceCents = Math.round(Number(price.replace(",", ".")) * 100);
    if (!Number.isSafeInteger(hourlyPriceCents)) {
      throw new Error("Der Stundenpreis ist zu groß.");
    }
    return { name, hourlyPriceCents };
  },
  toFormValues: ({ name, hourlyPriceCents }) => ({
    name,
    price: (hourlyPriceCents / 100).toFixed(2).replace(".", ","),
  }),
});

// ---------- Views ----------

async function loadStats() {
  const stats = await api("/api/stats");
  for (const node of document.querySelectorAll("[data-stat]")) {
    node.textContent = stats[node.dataset.stat];
  }
}

async function loadSkis() {
  const skis = await api("/api/skis");
  renderRows(byId("ski-rows"), skis.map((ski) => [
    String(ski.id),
    ski.name,
    ski.available ? badge("Verfügbar", "available") : badge("Ausgeliehen", "rented"),
    actionButton("Bearbeiten", () => skiForm.open(ski)),
  ]), "Noch keine Ski angelegt.");
}

async function loadCustomers() {
  const customers = await api("/api/customers");
  renderRows(byId("customer-rows"), customers.map((customer) => [
    String(customer.id),
    customer.name,
    customer.idType,
    customer.idNumber,
    actionButton("Bearbeiten", () => customerForm.open(customer)),
  ]), "Noch keine Personen registriert.");
}

async function loadTariffs() {
  const tariffs = await api("/api/tariffs");
  renderRows(byId("tariff-rows"), tariffs.map((tariff) => [
    String(tariff.id),
    tariff.name,
    noWrap(`${formatEuro(tariff.hourlyPriceCents)} / Std.`),
    actionButton("Bearbeiten", () => tariffForm.open(tariff)),
  ]), "Noch keine Tarife angelegt.");
}

function rentalCells(rental) {
  return [
    String(rental.id),
    rental.customerName,
    rental.skiName,
    noWrap(formatDate(rental.start)),
    noWrap(formatDate(rental.end)),
    rental.tariffName,
    noWrap(formatPrice(rental)),
  ];
}

async function returnRental(rental, button) {
  if (!confirm(`Rückgabe von „${rental.skiName}“ (${rental.customerName}) bestätigen?`)) {
    return;
  }
  button.disabled = true;
  try {
    await api(`/api/rentals/${rental.id}/return`, { method: "POST" });
    showMessage(`„${rental.skiName}“ wurde zurückgegeben und ist wieder verfügbar.`, "success");
  } catch (error) {
    showMessage(error.message, "error");
  }
  await refreshView();
  restoreFocus();
}

async function loadRentals() {
  const rentals = await api("/api/rentals");
  const active = rentals.filter((rental) => rental.active);
  const completed = rentals.filter((rental) => !rental.active).reverse();

  byId("active-rental-count").textContent = active.length;
  byId("completed-rental-count").textContent = completed.length;

  renderRows(byId("active-rental-rows"), active.map((rental) => [
    ...rentalCells(rental),
    actionButton("Zurückgeben", (button) => returnRental(rental, button)),
  ]), "Zurzeit sind keine Ski ausgeliehen.");

  renderRows(byId("completed-rental-rows"), completed.map((rental) => [
    ...rentalCells(rental),
    noWrap(formatDate(rental.returnedAt)),
  ]), "Noch keine abgeschlossenen Ausleihen.");
}

function showRentalFilter(filter) {
  for (const button of document.querySelectorAll("[data-rental-filter]")) {
    const selected = button.dataset.rentalFilter === filter;
    button.setAttribute("aria-pressed", selected);
    byId(`rentals-${button.dataset.rentalFilter}`).hidden = !selected;
  }
}

// ---------- Rental form ----------

const customerLabel = (customer) => `${customer.name} (${customer.idNumber})`;

async function loadRentalOptions() {
  const [customers, skis, tariffs] = await Promise.all([
    api("/api/customers"),
    api("/api/skis"),
    api("/api/tariffs"),
  ]);
  const { elements } = byId("rental-form");
  fillSelect(elements.customerId, customers, customerLabel, "Keine Personen registriert");
  fillSelect(elements.skiId, skis.filter((ski) => ski.available), (ski) => `${ski.id}: ${ski.name}`, "Kein Ski verfügbar");
  fillSelect(elements.tariffId, tariffs, (tariff) => `${tariff.name} (${formatEuro(tariff.hourlyPriceCents)} / Std.)`, "Keine Tarife angelegt");
}

// setStartToNow fills in the current local time as datetime-local expects it.
function setStartToNow() {
  const now = new Date();
  now.setMinutes(now.getMinutes() - now.getTimezoneOffset());
  byId("rental-form").elements.start.value = now.toISOString().slice(0, 16);
}

let latestQuote = 0;

// updatePricePreview asks the backend for the price. Only the newest answer is
// shown, because responses can arrive out of order while the user is typing.
async function updatePricePreview() {
  const quote = ++latestQuote;
  const { tariffId, durationHours } = byId("rental-form").elements;
  const preview = byId("price-preview");

  if (!tariffId.value || !durationHours.checkValidity()) {
    preview.textContent = "Bitte Tarif wählen und gültige Dauer eingeben";
    return;
  }

  preview.textContent = "Wird berechnet …";
  let text;
  try {
    const params = new URLSearchParams({ tariffId: tariffId.value, hours: durationHours.value });
    text = formatPrice(await api(`/api/price-quote?${params}`));
  } catch (error) {
    text = error.message;
  }
  if (quote === latestQuote) {
    preview.textContent = text;
  }
}

// openRentalForm loads current choices first, so a rented ski is never offered.
async function openRentalForm() {
  const form = byId("rental-form");
  form.reset();
  byId("customer-added").hidden = true;
  try {
    await loadRentalOptions();
  } catch (error) {
    showMessage(error.message, "error");
    return;
  }
  setStartToNow();
  updatePricePreview();
  openDialog(form);
}

// ---------- New person during a rental ----------
// The person form moves into the rental dialog, so its fields, validation and
// saving exist only once. Forms cannot be nested, so the rental form is hidden
// meanwhile, which keeps everything already entered.

const registeringCustomer = () => byId("rental-dialog").contains(byId("customer-form"));

function startRegistration() {
  const form = byId("customer-form");
  byId("customer-added").hidden = true;
  byId("rental-form").hidden = true;
  byId("rental-dialog").append(form);
  byId("rental-dialog").setAttribute("aria-labelledby", "customer-dialog-title");
  customerForm.open(null, selectNewCustomer);
  form.elements.name.focus();
}

function endRegistration() {
  if (registeringCustomer()) {
    byId("customer-dialog").append(byId("customer-form"));
    byId("rental-dialog").setAttribute("aria-labelledby", "rental-dialog-title");
    byId("rental-form").hidden = false;
  }
}

function cancelRegistration() {
  // While saving, the person may already exist, so the result decides what comes next.
  if (!byId("customer-form").querySelector("[type=submit]").disabled) {
    endRegistration();
    byId("new-rental-customer").focus();
  }
}

// selectNewCustomer uses the saved person directly. Reloading the list could
// fail, and the person would then look unsaved although they already exist.
async function selectNewCustomer(customer) {
  endRegistration();
  const select = byId("rental-form").elements.customerId;
  select.add(new Option(customerLabel(customer), customer.id));
  select.value = customer.id;
  byId("customer-added").hidden = false;
  select.focus();
  await refreshView();
}

function setupRentalForm() {
  const form = byId("rental-form");
  const dialog = byId("rental-dialog");
  form.addEventListener("input", updatePricePreview);
  byId("new-rental-customer").addEventListener("click", startRegistration);

  // Esc leaves only the person form, so the rental data stays. Browsers may
  // ignore preventDefault() on a repeated cancel event, but not on the key itself.
  const leaveRegistration = (event) => {
    if (registeringCustomer() && (event.type === "cancel" || event.key === "Escape")) {
      event.preventDefault();
      cancelRegistration();
    }
  };
  dialog.addEventListener("keydown", leaveRegistration);
  dialog.addEventListener("cancel", leaveRegistration);
  dialog.addEventListener("close", endRegistration);

  handleSubmit(form, async (values) => {
    try {
      await sendJSON("/api/rentals", "POST", {
        customerId: Number(values.customerId),
        skiId: Number(values.skiId),
        tariffId: Number(values.tariffId),
        start: new Date(values.start).toISOString(),
        durationHours: Number(values.durationHours),
      });
    } catch (error) {
      // The ski list may be out of date. The original error is the one to show.
      await Promise.allSettled([loadRentalOptions(), refreshView()]);
      throw error;
    }
    showRentalFilter("active");
    await finishDialog(form, "Ausleihe angelegt.");
  });
}

// ---------- Navigation ----------

const views = {
  dashboard: async () => {}, // Only needs the statistics, which always load.
  ski: loadSkis,
  personen: loadCustomers,
  ausleihen: loadRentals,
  tarife: loadTariffs,
};

function currentView() {
  const name = location.hash.slice(1);
  return Object.hasOwn(views, name) ? name : "dashboard";
}

// refreshView reloads the visible page and the dashboard statistics.
async function refreshView() {
  const name = currentView();
  try {
    await Promise.all([loadStats(), views[name]()]);
  } catch (error) {
    for (const tbody of byId(`view-${name}`).querySelectorAll("tbody")) {
      showTableNote(tbody, "Die Daten konnten nicht geladen werden.");
    }
    for (const node of document.querySelectorAll("[data-stat]")) {
      node.textContent = "–";
    }
    showMessage(error.message, "error");
  }
}

function showView() {
  const name = currentView();
  hideMessage();
  for (const dialog of document.querySelectorAll("dialog[open]")) {
    dialog.close();
  }

  for (const view of document.querySelectorAll(".view")) {
    view.hidden = view.id !== `view-${name}`;
  }
  for (const link of document.querySelectorAll("nav a")) {
    if (link.hash === `#${name}`) {
      link.setAttribute("aria-current", "page");
    } else {
      link.removeAttribute("aria-current");
    }
  }
  if (name === "ausleihen") {
    showRentalFilter("active");
  }
  window.scrollTo(0, 0);
  refreshView();
}

function setupButtons() {
  byId("new-ski").addEventListener("click", () => skiForm.open());
  byId("new-customer").addEventListener("click", () => customerForm.open());
  byId("new-tariff").addEventListener("click", () => tariffForm.open());
  for (const button of document.querySelectorAll("[data-new-rental]")) {
    button.addEventListener("click", openRentalForm);
  }
  for (const button of document.querySelectorAll("[data-cancel]")) {
    button.addEventListener("click", () => {
      if (registeringCustomer()) {
        cancelRegistration();
      } else {
        button.closest("dialog").close();
      }
    });
  }
  for (const button of document.querySelectorAll("[data-rental-filter]")) {
    button.addEventListener("click", () => showRentalFilter(button.dataset.rentalFilter));
  }
}

byId("message").addEventListener("click", hideMessage);
window.addEventListener("hashchange", showView);
setupButtons();
setupRentalForm();
showView();
