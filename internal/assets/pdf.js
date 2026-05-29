import * as pdfjsLib from "/assets/vendor/pdfjs/pdf.min.mjs";

const root = document.documentElement;
const main = document.getElementById("pdf-root");
const pagesEl = document.getElementById("pdf-pages");
const statusEl = document.getElementById("pdf-status");
const toast = document.getElementById("toast");

pdfjsLib.GlobalWorkerOptions.workerSrc = main.dataset.worker;

// ---- Theme ----------------------------------------------------------------

const THEME_KEY = "mdview-theme";
function applyTheme(t) {
  root.setAttribute("data-theme",
    (t === "light" || t === "dark") ? t : "auto");
}
applyTheme(localStorage.getItem(THEME_KEY) || "auto");

function toggleTheme() {
  const current = root.getAttribute("data-theme");
  const prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches;
  const effective = current === "auto" ? (prefersDark ? "dark" : "light") : current;
  const next = effective === "dark" ? "light" : "dark";
  applyTheme(next);
  localStorage.setItem(THEME_KEY, next);
  showToast("Theme: " + next);
}

// ---- Toast ----------------------------------------------------------------

let toastTimer;
function showToast(msg) {
  toast.textContent = msg;
  toast.hidden = false;
  toast.classList.add("show");
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => {
    toast.classList.remove("show");
    setTimeout(() => { toast.hidden = true; }, 200);
  }, 1500);
}

// ---- Rendering ------------------------------------------------------------

let renderToken = 0;

async function loadAndRender() {
  const token = ++renderToken;
  statusEl.textContent = "Loading…";
  statusEl.hidden = false;
  try {
    // Cache-bust on reload so the watcher event picks up new bytes.
    const url = main.dataset.file + "?v=" + Date.now();
    // canvasMaxAreaInBytes caps the size of pdf.js' internal scratch canvases.
    // Without it WebKit2GTK silently draws blank pixels for large DeviceRGB
    // JPEGs (e.g. scanned A4 pages around ~9 MP); the smaller cap routes
    // those images through pdf.js' resizing path which works in WebKit.
    const doc = await pdfjsLib.getDocument({ url, canvasMaxAreaInBytes: 16 * 1024 * 1024 }).promise;
    if (token !== renderToken) return false; // superseded
    pagesEl.replaceChildren();
    const dpr = window.devicePixelRatio || 1;
    for (let i = 1; i <= doc.numPages; i++) {
      if (token !== renderToken) return false;
      const page = await doc.getPage(i);
      const viewport = page.getViewport({ scale: 1.5 });
      const canvas = document.createElement("canvas");
      canvas.className = "pdf-page";
      canvas.width = Math.floor(viewport.width * dpr);
      canvas.height = Math.floor(viewport.height * dpr);
      canvas.style.width = viewport.width + "px";
      canvas.style.height = viewport.height + "px";
      const ctx = canvas.getContext("2d");
      ctx.scale(dpr, dpr);
      pagesEl.appendChild(canvas);
      await page.render({ canvasContext: ctx, viewport }).promise;
    }
    statusEl.hidden = true;
    return true;
  } catch (err) {
    console.error("PDF render failed", err);
    statusEl.textContent = "Failed to load PDF: " + err.message;
    statusEl.hidden = false;
    return false;
  }
}

// ---- Live reload ---------------------------------------------------------

function connectEvents() {
  try {
    const es = new EventSource("/api/events");
    es.addEventListener("reload", () => {
      const y = window.scrollY;
      loadAndRender().then((completed) => {
        if (completed) window.scrollTo(0, y);
      });
    });
    es.onerror = () => { /* auto-reconnects */ };
  } catch (err) {
    console.error("EventSource failed", err);
  }
}

// ---- Keyboard shortcuts --------------------------------------------------

document.addEventListener("keydown", (e) => {
  if (e.ctrlKey || e.metaKey) return;
  switch (e.key) {
    case "q":
    case "Escape":
      if (window.quit) window.quit();
      break;
    case "t": toggleTheme(); break;
    case "r":
      loadAndRender();
      showToast("Reloaded");
      break;
    case "j":
    case "ArrowDown":
      window.scrollBy({ top: 80, behavior: "smooth" });
      break;
    case "k":
    case "ArrowUp":
      window.scrollBy({ top: -80, behavior: "smooth" });
      break;
    case "g":
      window.scrollTo({ top: 0, behavior: "smooth" });
      break;
    case "G":
      window.scrollTo({ top: document.body.scrollHeight, behavior: "smooth" });
      break;
  }
});

// ---- Initial pass --------------------------------------------------------

loadAndRender();
connectEvents();
