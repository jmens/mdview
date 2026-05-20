(function () {
  "use strict";

  const root = document.documentElement;
  const content = document.getElementById("content");

  // ---- Theme ----------------------------------------------------------------

  const THEME_KEY = "mdview-theme";
  function applyTheme(t) {
    if (t === "light" || t === "dark") {
      root.setAttribute("data-theme", t);
    } else {
      root.setAttribute("data-theme", "auto");
    }
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
    renderMermaid();
  }

  // ---- External links via Go binding ---------------------------------------

  function isExternal(href) {
    return /^(https?:|mailto:)/i.test(href);
  }
  document.addEventListener("click", function (e) {
    const a = e.target.closest("a");
    if (!a) return;
    const href = a.getAttribute("href");
    if (!href) return;
    if (isExternal(href)) {
      e.preventDefault();
      if (window.openExternal) window.openExternal(href);
    }
  });

  // ---- KaTeX & Mermaid rendering -------------------------------------------

  function renderMath() {
    if (typeof renderMathInElement !== "function") return;
    try {
      renderMathInElement(content, {
        delimiters: [
          { left: "$$", right: "$$", display: true },
          { left: "$",  right: "$",  display: false },
          { left: "\\(", right: "\\)", display: false },
          { left: "\\[", right: "\\]", display: true },
        ],
        throwOnError: false,
      });
    } catch (err) {
      console.error("KaTeX render failed", err);
    }
  }

  function mermaidTheme() {
    const t = root.getAttribute("data-theme");
    if (t === "dark") return "dark";
    if (t === "light") return "default";
    return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "default";
  }

  let mermaidInitialized = false;
  function renderMermaid() {
    if (typeof mermaid === "undefined") return;
    mermaid.initialize({ startOnLoad: false, theme: mermaidTheme(), securityLevel: "loose" });
    mermaidInitialized = true;
    const nodes = content.querySelectorAll("pre.mermaid, pre > code.language-mermaid");
    const blocks = [];
    nodes.forEach(function (node) {
      if (node.tagName === "PRE") {
        blocks.push(node);
      } else {
        const pre = node.parentElement;
        const wrapper = document.createElement("div");
        wrapper.className = "mermaid";
        wrapper.textContent = node.textContent;
        pre.replaceWith(wrapper);
        blocks.push(wrapper);
      }
    });
    if (blocks.length > 0) {
      try {
        mermaid.run({ nodes: blocks });
      } catch (err) {
        console.error("Mermaid render failed", err);
      }
    }
  }

  // ---- Live reload ---------------------------------------------------------

  function reload() {
    fetch("/api/content")
      .then(function (r) { return r.json(); })
      .then(function (data) {
        const scrollY = window.scrollY;
        if (data.error) {
          content.innerHTML = '<div class="render-error">' +
            escapeHtml(data.error) + "</div>" + (data.html || "");
        } else {
          content.innerHTML = data.html || "";
        }
        if (data.title) document.title = data.title;
        renderMath();
        renderMermaid();
        window.scrollTo(0, scrollY);
      })
      .catch(function (err) {
        console.error("reload failed", err);
        showToast("Reload failed");
      });
  }

  function connectEvents() {
    try {
      const es = new EventSource("/api/events");
      es.addEventListener("reload", reload);
      es.onerror = function () {
        // EventSource auto-reconnects; nothing to do.
      };
    } catch (err) {
      console.error("EventSource failed", err);
    }
  }

  function escapeHtml(s) {
    return String(s)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#39;");
  }

  // ---- Toast ---------------------------------------------------------------

  const toast = document.getElementById("toast");
  let toastTimer;
  function showToast(msg) {
    toast.textContent = msg;
    toast.hidden = false;
    toast.classList.add("show");
    clearTimeout(toastTimer);
    toastTimer = setTimeout(function () {
      toast.classList.remove("show");
      setTimeout(function () { toast.hidden = true; }, 200);
    }, 1500);
  }

  // ---- Search --------------------------------------------------------------

  const searchBar = document.getElementById("search-bar");
  const searchInput = document.getElementById("search-input");
  const searchCount = document.getElementById("search-count");
  let searchHits = [];
  let searchIndex = -1;

  function clearHighlights() {
    content.querySelectorAll("mark.mdview-hit").forEach(function (m) {
      const text = document.createTextNode(m.textContent);
      m.replaceWith(text);
    });
    content.normalize();
    searchHits = [];
    searchIndex = -1;
    searchCount.textContent = "";
  }

  function highlight(term) {
    clearHighlights();
    if (!term) return;
    const re = new RegExp(term.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"), "gi");
    const walker = document.createTreeWalker(content, NodeFilter.SHOW_TEXT, {
      acceptNode: function (node) {
        if (!node.nodeValue || !node.nodeValue.trim()) return NodeFilter.FILTER_REJECT;
        const p = node.parentElement;
        if (p && /^(SCRIPT|STYLE|MARK)$/.test(p.tagName)) return NodeFilter.FILTER_REJECT;
        return re.test(node.nodeValue) ? NodeFilter.FILTER_ACCEPT : NodeFilter.FILTER_REJECT;
      },
    });

    const targets = [];
    let n;
    while ((n = walker.nextNode())) targets.push(n);

    targets.forEach(function (node) {
      const frag = document.createDocumentFragment();
      let last = 0;
      const text = node.nodeValue;
      re.lastIndex = 0;
      let m;
      while ((m = re.exec(text))) {
        if (m.index > last) frag.appendChild(document.createTextNode(text.slice(last, m.index)));
        const mark = document.createElement("mark");
        mark.className = "mdview-hit";
        mark.textContent = m[0];
        frag.appendChild(mark);
        last = m.index + m[0].length;
        if (m[0].length === 0) re.lastIndex++;
      }
      if (last < text.length) frag.appendChild(document.createTextNode(text.slice(last)));
      node.parentNode.replaceChild(frag, node);
    });

    searchHits = Array.from(content.querySelectorAll("mark.mdview-hit"));
    if (searchHits.length > 0) {
      searchIndex = 0;
      focusHit();
    } else {
      searchCount.textContent = "0";
    }
  }

  function focusHit() {
    searchHits.forEach(function (h) { h.classList.remove("active"); });
    if (searchIndex < 0 || searchIndex >= searchHits.length) return;
    const hit = searchHits[searchIndex];
    hit.classList.add("active");
    hit.scrollIntoView({ behavior: "smooth", block: "center" });
    searchCount.textContent = (searchIndex + 1) + " / " + searchHits.length;
  }

  function nextHit(dir) {
    if (searchHits.length === 0) return;
    searchIndex = (searchIndex + dir + searchHits.length) % searchHits.length;
    focusHit();
  }

  function openSearch() {
    searchBar.hidden = false;
    searchInput.focus();
    searchInput.select();
  }
  function closeSearch() {
    searchBar.hidden = true;
    clearHighlights();
    searchInput.value = "";
  }

  searchInput.addEventListener("input", function () { highlight(searchInput.value); });
  searchInput.addEventListener("keydown", function (e) {
    if (e.key === "Enter") {
      e.preventDefault();
      nextHit(e.shiftKey ? -1 : 1);
    } else if (e.key === "Escape") {
      e.preventDefault();
      closeSearch();
    }
  });
  document.getElementById("search-next").addEventListener("click", function () { nextHit(1); });
  document.getElementById("search-prev").addEventListener("click", function () { nextHit(-1); });
  document.getElementById("search-close").addEventListener("click", closeSearch);

  // ---- Keyboard shortcuts --------------------------------------------------

  document.addEventListener("keydown", function (e) {
    if (e.target === searchInput) return;
    if (e.ctrlKey || e.metaKey) {
      if (e.key === "f") {
        e.preventDefault();
        openSearch();
        return;
      }
      return;
    }
    switch (e.key) {
      case "q":
      case "Escape":
        if (!searchBar.hidden) { closeSearch(); break; }
        if (window.quit) window.quit();
        break;
      case "t": toggleTheme(); break;
      case "r": reload(); showToast("Reloaded"); break;
      case "/":
        e.preventDefault();
        openSearch();
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
      case "n":
        if (searchHits.length > 0) nextHit(1);
        break;
      case "N":
        if (searchHits.length > 0) nextHit(-1);
        break;
    }
  });

  // ---- Initial pass --------------------------------------------------------

  renderMath();
  renderMermaid();
  connectEvents();
})();
