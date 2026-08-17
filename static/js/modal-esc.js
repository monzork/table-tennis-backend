// Closes any visible modal/popup overlay when Escape is pressed. Covers
// every modal in the app generically (they all share the "fixed inset-0"
// full-screen overlay pattern), including ones toggled by an inline
// onclick/hx-on:click rather than the openModal()/closeModal() helpers.
document.addEventListener('keydown', function (e) {
    if (e.key !== 'Escape') return;
    document.querySelectorAll('.fixed.inset-0:not(.hidden)').forEach(function (overlay) {
        if (overlay.id === 'global-spinner' || overlay.classList.contains('htmx-indicator')) return;
        overlay.classList.add('hidden');
        overlay.setAttribute('aria-hidden', 'true');
    });
});
