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

// handleSubmit checks that the form is complete, runs action and reports
// errors. It only checks the input format; the backend makes the final decision.
function handleSubmit(form, action) {
  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    hideMessage();

    const invalidField = [...form.elements].find((field) => !field.checkValidity());
    if (invalidField) {
      showMessage(invalidField.dataset.error, "error");
      invalidField.focus();
      return;
    }

    const submitButton = form.querySelector("[type=submit]");
    submitButton.disabled = true;
    try {
      await action(Object.fromEntries(new FormData(form)));
    } catch (error) {
      showMessage(error.message, "error");
    } finally {
      submitButton.disabled = false;
    }
  });
}

// createEditableForm connects a form that both creates and edits an item.
// toBody turns form values into the request body, and toFormValues does the
// opposite for the item being edited.
function createEditableForm({ formId, path, noun, toBody = (values) => values, toFormValues = (item) => item }) {
  const form = byId(formId);
  const title = form.querySelector("h2");
  const submitButton = form.querySelector("[type=submit]");
  const cancelButton = form.querySelector("[data-cancel]");
  const createTitle = title.textContent;
  const createLabel = submitButton.textContent;
  let editingId = null;

  function stopEditing() {
    if (editingId === null) {
      return;
    }
    editingId = null;
    form.reset();
    form.classList.remove("editing");
    title.textContent = createTitle;
    submitButton.textContent = createLabel;
    cancelButton.hidden = true;
  }

  function startEditing(item) {
    editingId = item.id;
    const values = toFormValues(item);
    for (const field of form.elements) {
      if (field.name) {
        field.value = values[field.name];
      }
    }
    form.classList.add("editing");
    title.textContent = `${noun} bearbeiten`;
    submitButton.textContent = "Änderungen speichern";
    cancelButton.hidden = false;
    form.scrollIntoView({ block: "nearest" });
    form.elements[0].focus();
  }

  cancelButton.addEventListener("click", () => {
    stopEditing();
    form.elements[0].focus(); // The focused cancel button is now hidden.
  });

  handleSubmit(form, async (values) => {
    const body = toBody(values);
    if (editingId === null) {
      await sendJSON(path, "POST", body);
      form.reset();
      showMessage(`${noun} angelegt.`, "success");
    } else {
      await sendJSON(`${path}/${editingId}`, "PUT", body);
      stopEditing();
      showMessage("Änderungen gespeichert.", "success");
    }
    await refreshView();
  });

  return { startEditing, stopEditing };
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
    actionButton("Bearbeiten", () => skiForm.startEditing(ski)),
  ]), "Noch keine Ski angelegt.");
}

async function loadCustomers() {
  const customers = await api("/api/customers");
  renderRows(byId("customer-rows"), customers.map((customer) => [
    String(customer.id),
    customer.name,
    customer.idType,
    customer.idNumber,
    actionButton("Bearbeiten", () => customerForm.startEditing(customer)),
  ]), "Noch keine Personen registriert.");
}

async function loadTariffs() {
  const tariffs = await api("/api/tariffs");
  renderRows(byId("tariff-rows"), tariffs.map((tariff) => [
    String(tariff.id),
    tariff.name,
    noWrap(`${formatEuro(tariff.hourlyPriceCents)} / Std.`),
    actionButton("Bearbeiten", () => tariffForm.startEditing(tariff)),
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
}

async function loadRentals() {
  const [rentals, customers, skis, tariffs] = await Promise.all([
    api("/api/rentals"),
    api("/api/customers"),
    api("/api/skis"),
    api("/api/tariffs"),
  ]);

  const { elements } = byId("rental-form");
  fillSelect(elements.customerId, customers, (customer) => `${customer.name} (${customer.idNumber})`, "Keine Personen registriert");
  fillSelect(elements.skiId, skis.filter((ski) => ski.available), (ski) => `${ski.id}: ${ski.name}`, "Kein Ski verfügbar");
  fillSelect(elements.tariffId, tariffs, (tariff) => `${tariff.name} (${formatEuro(tariff.hourlyPriceCents)} / Std.)`, "Keine Tarife angelegt");
  if (!elements.start.value) {
    setStartToNow();
  }
  updatePricePreview();

  const active = rentals.filter((rental) => rental.active);
  const completed = rentals.filter((rental) => !rental.active).reverse();

  renderRows(byId("active-rental-rows"), active.map((rental) => [
    ...rentalCells(rental),
    badge("Aktiv", "active"),
    actionButton("Zurückgeben", (button) => returnRental(rental, button)),
  ]), "Zurzeit sind keine Ski ausgeliehen.");

  renderRows(byId("completed-rental-rows"), completed.map((rental) => [
    ...rentalCells(rental),
    badge("Zurückgegeben", "completed"),
    noWrap(formatDate(rental.returnedAt)),
  ]), "Noch keine abgeschlossenen Ausleihen.");
}

// ---------- Rental form ----------

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

function setupRentalForm() {
  const form = byId("rental-form");
  form.addEventListener("input", updatePricePreview);

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
      await refreshView(); // The ski list may be out of date.
      throw error;
    }
    form.reset();
    showMessage("Ausleihe angelegt.", "success");
    await refreshView();
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
  skiForm.stopEditing();
  customerForm.stopEditing();
  tariffForm.stopEditing();

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
    setStartToNow(); // A start time from an earlier visit would be out of date.
  }
  window.scrollTo(0, 0);
  refreshView();
}

byId("message").addEventListener("click", hideMessage);
window.addEventListener("hashchange", showView);
setupRentalForm();
showView();
