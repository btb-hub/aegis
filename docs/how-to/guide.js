(function () {
  document.querySelectorAll("pre.cmd").forEach(function (pre) {
    var source = pre.querySelector("code") || pre;
    var btn = document.createElement("button");
    btn.type = "button";
    btn.className = "copy";
    btn.textContent = "copy";
    btn.addEventListener("click", function () {
      navigator.clipboard.writeText(source.textContent.trim()).then(function () {
        btn.textContent = "copied";
        btn.classList.add("ok");
        setTimeout(function () {
          btn.textContent = "copy";
          btn.classList.remove("ok");
        }, 1400);
      });
    });
    pre.appendChild(btn);
  });
})();
