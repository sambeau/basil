// Client-side behaviour for herbaceous.net: syntax highlighting, the
// light/dark theme toggle, the docs search box, and the homepage's
// "latest release" line. Loaded at the end of <body> on every page
// (see layout.pars). The theme is *applied* by a tiny inline script in
// <head> so pages don't flash the wrong colours; this file only toggles it.

hljs.highlightAll();

// Theme toggle
(function () {
  var btn = document.getElementById("theme-toggle");
  function paint() {
    var t = document.documentElement.getAttribute("data-theme");
    btn.textContent = t === "dark" ? "☀" : "☾";
  }
  btn.addEventListener("click", function () {
    var t = document.documentElement.getAttribute("data-theme") === "dark" ? "light" : "dark";
    document.documentElement.setAttribute("data-theme", t);
    localStorage.setItem("theme", t);
    paint();
  });
  paint();
})();

// Site search: fetches the build's search-index.json on first focus and
// scores pages client-side (title > headings > body).
(function () {
  var input = document.getElementById("site-search");
  var list = document.getElementById("search-results");
  if (!input) { return; }
  var root = document.body.getAttribute("data-root") || "";
  var index = null;

  function load() {
    if (index) { return; }
    fetch(root + "search-index.json")
      .then(function (r) { return r.json(); })
      .then(function (data) { index = data; })
      .catch(function () { index = []; });
  }

  function score(entry, terms) {
    var total = 0;
    for (var i = 0; i < terms.length; i++) {
      var t = terms[i];
      // cheap stemming: "passkeys" should find "passkey"
      var stem = t.length > 3 && t.charAt(t.length - 1) === "s" ? t.slice(0, -1) : t;
      var s = 0;
      if (entry.title.toLowerCase().indexOf(stem) !== -1) { s += 10; }
      if (entry.headings.toLowerCase().indexOf(stem) !== -1) { s += 4; }
      if (entry.text.toLowerCase().indexOf(stem) !== -1) { s += 1; }
      if (s === 0) { return 0; } // every term must match somewhere
      total += s;
    }
    return total;
  }

  function render(results) {
    list.textContent = "";
    if (results.length === 0) {
      list.hidden = true;
      return;
    }
    for (var i = 0; i < results.length; i++) {
      var e = results[i];
      var li = document.createElement("li");
      var a = document.createElement("a");
      a.href = root + e.path;
      a.textContent = e.title;
      var small = document.createElement("small");
      small.textContent = " " + e.section;
      a.appendChild(small);
      li.appendChild(a);
      list.appendChild(li);
    }
    list.hidden = false;
  }

  input.addEventListener("focus", load);
  input.addEventListener("input", function () {
    var q = input.value.trim().toLowerCase();
    if (!index || q.length < 2) {
      list.hidden = true;
      return;
    }
    var terms = q.split(/\s+/);
    var results = index
      .map(function (e) { return {e: e, s: score(e, terms)}; })
      .filter(function (r) { return r.s > 0; })
      .sort(function (a, b) { return b.s - a.s; })
      .slice(0, 8)
      .map(function (r) { return r.e; });
    render(results);
  });
  input.addEventListener("keydown", function (ev) {
    if (ev.key === "Enter") {
      var first = list.querySelector("a");
      if (first) { window.location.href = first.href; }
    }
    if (ev.key === "Escape") {
      list.hidden = true;
      input.blur();
    }
  });
  document.addEventListener("click", function (ev) {
    if (!input.contains(ev.target) && !list.contains(ev.target)) {
      list.hidden = true;
    }
  });
})();

// Homepage only: fill in the "Latest release:" line from the GitHub API
(function () {
  var span = document.getElementById("latest-version");
  if (!span) { return; }
  fetch("https://api.github.com/repos/sambeau/basil/releases?per_page=1")
    .then(function (r) { return r.json(); })
    .then(function (rs) {
      if (rs && rs[0]) {
        span.textContent = "Latest release: " + rs[0].tag_name + ".";
      }
    })
    .catch(function () {});
})();
