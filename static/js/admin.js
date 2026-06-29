/**
 * admin.js — Extracted JavaScript for the Open TDM Nicaragua admin portal.
 * Previously inline in layouts/admin.html.
 */

// ── Theme Switcher ────────────────────────────────────────────────────────────
const THEME_KEY = 'club-theme';

function applyTheme(theme) {
    const isProj = (theme === 'proj');
    document.body.classList.toggle('proj', isProj);
    const label = document.getElementById('theme-label');
    const iconProj = document.getElementById('theme-icon-proj');
    const iconDark = document.getElementById('theme-icon-dark');
    if (label) label.textContent = isProj ? '← Admin Mode' : 'Projector';
    if (iconProj) iconProj.classList.toggle('hidden', !isProj);
    if (iconDark) iconDark.classList.toggle('hidden', isProj);
}

function toggleTheme() {
    const next = document.body.classList.contains('proj') ? 'dark' : 'proj';
    localStorage.setItem(THEME_KEY, next);
    applyTheme(next);
}

// Restore saved preference immediately (before first paint)
applyTheme(localStorage.getItem(THEME_KEY) || 'dark');

// ── Participant Selection Sync ────────────────────────────────────────────────
// Synchronise search/filter checked state for player selection cards.
document.addEventListener('change', function(e) {
    if (e.target.classList.contains('participant-checkbox')) {
        const grid = e.target.closest('#player-selection-grid, #edit-player-selection-grid');
        if (!grid) return;

        const isEdit = grid.id === 'edit-player-selection-grid';
        const containerId = isEdit ? 'edit-selected-participants-hidden' : 'selected-participants-hidden';
        const container = document.getElementById(containerId);
        if (!container) return;

        const val = e.target.value;
        const existing = container.querySelector(`input[value="${val}"]`);

        if (e.target.checked) {
            if (!existing) {
                const hidden = document.createElement('input');
                hidden.type = 'hidden';
                hidden.name = 'participant_ids[]';
                hidden.value = val;
                container.appendChild(hidden);
            }
        } else {
            if (existing) {
                existing.remove();
            }
        }
    }
});

// After HTMX swaps the player grid, re-sync checked state from the hidden inputs.
document.addEventListener('htmx:afterSwap', function(e) {
    const target = e.target;
    if (target && (target.id === 'player-selection-grid' || target.id === 'edit-player-selection-grid')) {
        const isEdit = target.id === 'edit-player-selection-grid';
        const containerId = isEdit ? 'edit-selected-participants-hidden' : 'selected-participants-hidden';
        const container = document.getElementById(containerId);
        if (!container) return;

        const checkboxes = target.querySelectorAll('.participant-checkbox');
        checkboxes.forEach(cb => {
            const val = cb.value;
            const existing = container.querySelector(`input[value="${val}"]`);
            if (cb.checked) {
                if (!existing) {
                    const hidden = document.createElement('input');
                    hidden.type = 'hidden';
                    hidden.name = 'participant_ids[]';
                    hidden.value = val;
                    container.appendChild(hidden);
                }
            } else {
                if (existing) {
                    existing.remove();
                }
            }
        });
    }
});

// ── Modal Helpers ─────────────────────────────────────────────────────────────

/**
 * Opens a modal by ID, sets aria-hidden=false, and traps focus inside it.
 * @param {string} modalId
 */
function openModal(modalId) {
    const modal = document.getElementById(modalId);
    if (!modal) return;
    modal.classList.remove('hidden');
    modal.setAttribute('aria-hidden', 'false');

    // Focus the first focusable element inside the modal
    const focusable = modal.querySelectorAll(
        'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
    );
    if (focusable.length) focusable[0].focus();

    // Trap focus inside modal
    modal.__focusTrap = function(e) {
        if (e.key !== 'Tab') return;
        const first = focusable[0];
        const last = focusable[focusable.length - 1];
        if (e.shiftKey) {
            if (document.activeElement === first) { e.preventDefault(); last.focus(); }
        } else {
            if (document.activeElement === last) { e.preventDefault(); first.focus(); }
        }
    };
    modal.addEventListener('keydown', modal.__focusTrap);

    // Close on Escape key
    modal.__escListener = function(e) {
        if (e.key === 'Escape') closeModal(modalId);
    };
    document.addEventListener('keydown', modal.__escListener);
}

/**
 * Closes a modal by ID and restores focus to the previously focused element.
 * @param {string} modalId
 */
function closeModal(modalId) {
    const modal = document.getElementById(modalId);
    if (!modal) return;
    modal.classList.add('hidden');
    modal.setAttribute('aria-hidden', 'true');
    if (modal.__focusTrap) {
        modal.removeEventListener('keydown', modal.__focusTrap);
        modal.__focusTrap = null;
    }
    if (modal.__escListener) {
        document.removeEventListener('keydown', modal.__escListener);
        modal.__escListener = null;
    }
}
