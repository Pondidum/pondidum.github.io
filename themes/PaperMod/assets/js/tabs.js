function switchTab(group, name) {
  const all = document.querySelectorAll(`[data-tab-group="${group}"]`);
  const target = document.querySelectorAll(
    `[data-tab-group="${group}"][data-tab-item="${name}"]`
  );

  all.forEach((e) => e.classList.remove("active"));
  target.forEach((e) => e.classList.add("active"));
}
