// Terminal demo: types the command, then prints the real output.
(function () {
  var demos = window.JUSTZ_DEMOS, body = document.getElementById("term-body"),
      title = document.getElementById("term-title"), term = document.getElementById("term"), tabs = document.querySelectorAll("#tabs button"),
      reduce = matchMedia("(prefers-reduced-motion: reduce)").matches, run = 0, timer;

  function esc(s) { return s.replace(/&/g, "&amp;").replace(/</g, "&lt;"); }
  function color(line) {
    var l = esc(line);
    l = l.replace(/^(\s*)(error)\b/, '$1<span class="e">$2</span>')
         .replace(/^(\s*)(warning)\b/, '$1<span class="w">$2</span>')
         .replace(/^(\s*)(paste:|fix:)/, '$1<span class="p">$2</span>')
         .replace(/(\[[a-z0-9_.-]+\])\s*$/, '<span class="dim">$1</span>')
         .replace(/(#.*)$/, '<span class="dim">$1</span>');
    return l;
  }
  function paint(d, typed, full) {
    var head = '<span class="p">$</span> ' + esc(d.cmd.slice(0, typed));
    var out = full ? "\n" + d.out.split("\n").map(color).join("\n") : '<span class="cur"></span>';
    body.innerHTML = head + out + (full ? '\n\n<span class="p">$</span> <span class="cur"></span>' : "");
  }
  function play(i) {
    var d = demos[i], id = ++run; clearTimeout(timer);
    title.textContent = d.cmd;
    tabs.forEach(function (t, k) { t.classList.toggle("on", k === i); t.setAttribute("aria-selected", k === i); t.tabIndex = k === i ? 0 : -1; });
    term.setAttribute("aria-label", "Terminal: " + d.cmd + "\n" + d.out);
    if (reduce) return paint(d, d.cmd.length, true);
    var n = 0;
    (function type() {
      if (id !== run) return;
      paint(d, n, false);
      if (n++ < d.cmd.length) timer = setTimeout(type, 45);
      else timer = setTimeout(function () { if (id === run) paint(d, d.cmd.length, true); }, 350);
    })();
  }
  // The demos array also holds the JSON sample, which has no tab: rotate over the tabs only.
  var cur = 0, auto = !reduce, loop, hover = false, visible = true;
  function schedule() {
    clearInterval(loop);
    if (!auto) return;
    loop = setInterval(function () {
      if (hover || !visible || document.hidden) return;
      cur = (cur + 1) % tabs.length; play(cur);
    }, 7500);
  }
  function pick(i) { cur = i; auto = false; clearInterval(loop); play(cur); tabs[cur].focus(); }
  tabs.forEach(function (t, k) {
    t.addEventListener("click", function () { pick(k); });
    t.addEventListener("keydown", function (e) {
      var d = e.key === "ArrowRight" ? 1 : e.key === "ArrowLeft" ? -1 : 0;
      if (d) { e.preventDefault(); pick((k + d + tabs.length) % tabs.length); }
    });
  });
  [term, document.getElementById("tabs")].forEach(function (el) {
    el.addEventListener("pointerenter", function () { hover = true; });
    el.addEventListener("pointerleave", function () { hover = false; });
    el.addEventListener("focusin", function () { hover = true; });
    el.addEventListener("focusout", function () { hover = false; });
  });
  if ("IntersectionObserver" in window) {
    new IntersectionObserver(function (es) { visible = es[0].isIntersecting; }).observe(term);
  }
  play(0); schedule();
})();

// Agents section: the JSON output of the last demo, with keys and strings colored.
(function () {
  var el = document.getElementById("json-body"), d = window.JUSTZ_DEMOS[window.JUSTZ_DEMOS.length - 1];
  if (!el) return;
  var html = d.out.replace(/&/g, "&amp;").replace(/</g, "&lt;")
    .replace(/("(?:[^"\\]|\\.)*")(\s*:)?/g, function (m, str, colon) {
      return '<span class="' + (colon ? "k" : "s") + '">' + str + "</span>" + (colon || "");
    });
  el.innerHTML = '<span class="dim">$ ' + d.cmd + "</span>\n" + html;
})();
