(function () {
  var reduce = matchMedia("(prefers-reduced-motion: reduce)").matches;
  var $ = function (s, r) { return (r || document).querySelector(s); };
  var $$ = function (s, r) { return Array.prototype.slice.call((r || document).querySelectorAll(s)); };

  // The sections fade in on scroll. They are hidden only from here on, so a
  // page whose script did not load shows everything.
  document.documentElement.classList.add("js");

  // Mouse-follow glow and per-card spotlight.
  addEventListener("pointermove", function (e) {
    document.documentElement.style.setProperty("--mx", e.clientX + "px");
    document.documentElement.style.setProperty("--my", e.clientY + "px");
  });
  $$(".spot").forEach(function (el) {
    el.addEventListener("pointermove", function (e) {
      var r = el.getBoundingClientRect();
      el.style.setProperty("--x", e.clientX - r.left + "px");
      el.style.setProperty("--y", e.clientY - r.top + "px");
    });
  });

  // Scroll reveal.
  var io = "IntersectionObserver" in window && new IntersectionObserver(function (es) {
    es.forEach(function (e) { if (e.isIntersecting) { e.target.classList.add("in"); io.unobserve(e.target); } });
  }, { threshold: 0.12 });
  $$(".reveal").forEach(function (el) { io ? io.observe(el) : el.classList.add("in"); });

  // Copy buttons.
  $$(".install").forEach(function (box) {
    var btn = $("button", box), status = $("#copy-status");
    // The button has an aria-label, so its text is not announced: say the
    // result in the status line, which a screen reader reads.
    var reset;
    var say = function (text, ms) {
      btn.textContent = text; status.textContent = text + ": " + box.dataset.copy;
      clearTimeout(reset);
      reset = setTimeout(function () { btn.textContent = "Copy"; status.textContent = ""; }, ms);
    };
    btn.addEventListener("click", function () {
      var done = function () { say("Copied", 1500); };
      var fail = function () {
        var r = document.createRange(); r.selectNodeContents($("code", box));
        var sel = getSelection(); sel.removeAllRanges(); sel.addRange(r);
        say("Press Ctrl+C", 2500);
      };
      if (navigator.clipboard) navigator.clipboard.writeText(box.dataset.copy).then(done, fail); else fail();
    });
  });

  // Particle grid background.
  var cv = $("#bg"), ctx = cv.getContext("2d"), pts = [], W, H;
  var rgb, rgb2;
  function colors() {
    var st = getComputedStyle(document.documentElement);
    rgb = st.getPropertyValue("--particle").trim(); rgb2 = st.getPropertyValue("--particle2").trim();
  }
  colors();
  function size() {
    W = cv.width = innerWidth; H = cv.height = innerHeight;
    pts = [];
    var n = Math.min(70, Math.floor(W * H / 22000));
    for (var i = 0; i < n; i++) pts.push({ x: Math.random() * W, y: Math.random() * H, vx: (Math.random() - .5) * .25, vy: (Math.random() - .5) * .25 });
  }
  function frame() {
    ctx.clearRect(0, 0, W, H);
    pts.forEach(function (p, i) {
      p.x += p.vx; p.y += p.vy;
      if (p.x < 0 || p.x > W) p.vx *= -1;
      if (p.y < 0 || p.y > H) p.vy *= -1;
      ctx.fillStyle = "rgba(" + rgb + ",.55)";
      ctx.fillRect(p.x, p.y, 2, 2);
      for (var j = i + 1; j < pts.length; j++) {
        var q = pts[j], d = Math.hypot(p.x - q.x, p.y - q.y);
        if (d < 140) { ctx.strokeStyle = "rgba(" + rgb2 + "," + (.16 * (1 - d / 140)) + ")"; ctx.beginPath(); ctx.moveTo(p.x, p.y); ctx.lineTo(q.x, q.y); ctx.stroke(); }
      }
    });
    if (!reduce) requestAnimationFrame(frame);
  }
  // Theme toggle: dark by default, the OS preference when unset, the visitor's pick when stored.
  var root = document.documentElement, tbtn = $(".theme");
  tbtn.hidden = false;
  function theme() { return root.dataset.theme || (matchMedia("(prefers-color-scheme: light)").matches ? "light" : "dark"); }
  function sync() { tbtn.textContent = theme() === "light" ? "\u263E" : "\u2600"; tbtn.setAttribute("aria-label", "Switch to " + (theme() === "light" ? "dark" : "light") + " theme"); colors(); if (reduce) frame(); }
  try { var saved = localStorage.getItem("theme"); if (saved) root.dataset.theme = saved; } catch (e) {}
  tbtn.addEventListener("click", function () {
    root.dataset.theme = theme() === "light" ? "dark" : "light";
    try { localStorage.setItem("theme", root.dataset.theme); } catch (e) {}
    sync();
  });
  matchMedia("(prefers-color-scheme: light)").addEventListener("change", sync);
  sync();

  size(); frame(); addEventListener("resize", function () { size(); if (reduce) frame(); });
})();
