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
  function html(d, typed, full) {
    var head = '<span class="p">$</span> ' + esc(d.cmd.slice(0, typed));
    var out = full ? "\n" + d.out.split("\n").map(color).join("\n") : '<span class="cur"></span>';
    return head + out + (full ? '\n\n<span class="p">$</span> <span class="cur"></span>' : "");
  }
  function paint(d, typed, full) { body.innerHTML = html(d, typed, full); }
  function play(i) {
    var d = demos[i], id = ++run; clearTimeout(timer);
    title.textContent = d.cmd;
    tabs.forEach(function (t, k) { t.classList.toggle("on", k === i); t.setAttribute("aria-selected", k === i); t.tabIndex = k === i ? 0 : -1; });
    term.setAttribute("aria-labelledby", tabs[i].id);
    if (reduce) return paint(d, d.cmd.length, true);
    var n = 0;
    (function type() {
      if (id !== run) return;
      paint(d, n, false);
      if (n++ < d.cmd.length) timer = setTimeout(type, 45);
      else timer = setTimeout(function () { if (id === run) paint(d, d.cmd.length, true); }, 350);
    })();
  }
  // The terminal is as tall as its tallest demo at this width, so a change of
  // tab does not move the page under the reader. The demos are measured in a
  // hidden copy, and only when the width changed, so the terminal on screen,
  // its animation and a selection in it are left alone.
  var fitted = -1;
  function fit() {
    var w = body.clientWidth, max = 0;
    if (!w || w === fitted) return;
    fitted = w;
    var probe = body.cloneNode(false);
    probe.removeAttribute("id");
    probe.style.cssText = "position:absolute;visibility:hidden;min-height:0;width:" + w + "px";
    term.appendChild(probe);
    for (var i = 0; i < tabs.length; i++) { probe.innerHTML = html(demos[i], demos[i].cmd.length, true); max = Math.max(max, probe.offsetHeight); }
    term.removeChild(probe);
    body.style.minHeight = max + "px";
  }
  // The demos array also holds the JSON sample, which has no tab: rotate over the tabs only.
  // The rotation stops for good when the visitor picks a tab or pauses it, and
  // it waits while the pointer or the focus is on the demo.
  var cur = 0, auto = !reduce, loop, hover = false, focused = false, visible = true, pause = document.getElementById("demo-pause");
  function schedule() {
    clearInterval(loop);
    pause.textContent = auto ? "Pause the demo" : "Play the demo";
    if (!auto) return;
    loop = setInterval(function () {
      if (hover || focused || !visible || document.hidden) return;
      cur = (cur + 1) % tabs.length; play(cur);
    }, 7500);
  }
  function pick(i) { cur = i; auto = false; schedule(); play(cur); tabs[cur].focus(); }
  tabs.forEach(function (t, k) {
    t.addEventListener("click", function () { pick(k); });
    t.addEventListener("keydown", function (e) {
      var to = e.key === "ArrowRight" ? k + 1 : e.key === "ArrowLeft" ? k - 1 : e.key === "Home" ? 0 : e.key === "End" ? tabs.length - 1 : null;
      if (to !== null) { e.preventDefault(); pick((to + tabs.length) % tabs.length); }
    });
  });
  // With reduced motion nothing rotates, so there is nothing to pause.
  if (!reduce) {
    pause.hidden = false;
    pause.addEventListener("click", function () { auto = !auto; schedule(); });
  }
  [term, document.getElementById("tabs")].forEach(function (el) {
    el.addEventListener("pointerenter", function () { hover = true; });
    el.addEventListener("pointerleave", function () { hover = false; });
    el.addEventListener("focusin", function () { focused = true; });
    el.addEventListener("focusout", function () { focused = false; });
  });
  if ("IntersectionObserver" in window) {
    new IntersectionObserver(function (es) { visible = es[0].isIntersecting; }).observe(term);
  }
  // The demo is hidden in the page until this script can fill it.
  term.hidden = false; document.getElementById("tabs").hidden = false;
  fit(); addEventListener("resize", fit);
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
  el.parentNode.hidden = false;
})();
