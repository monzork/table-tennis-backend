/**
 * admin.js — Extracted JavaScript for the Open TDM Nicaragua admin portal.
 * Previously inline in layouts/admin.html.
 */

// ── Theme Switcher ────────────────────────────────────────────────────────────
var THEME_KEY = 'club-theme';

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

function printQRCode() {
    const src = document.getElementById('qr-image').src.replace(/size=\d+/, 'size=1000');
    const printWindow = window.open('', '', 'width=800,height=600');
    
    let html = `
        <html>
            <head>
                <title>Print QR</title>
                <style>
                    body { font-family: sans-serif; margin: 0; padding: 0; text-align: center; }
                    .page { page-break-after: always; height: 100vh; display: flex; flex-direction: column; align-items: center; justify-content: center; }
                    .qr-img { width: 80vmin; height: 80vmin; max-width: 800px; max-height: 800px; }
                    .table-text { font-size: 40vmin; font-weight: bold; margin: 0; line-height: 1; }
                    .label { font-size: 6vmin; margin-bottom: 20px; color: #555; text-transform: uppercase; letter-spacing: 2px; }
                    @media print {
                        @page { margin: 0; }
                        body { -webkit-print-color-adjust: exact; }
                    }
                </style>
            </head>
            <body>
                <div class="page">
                    <div class="label">Escanear para Anotar</div>
                    <img class="qr-img" src="${src}" onload="window.print();window.close();" />
                </div>
    `;
    
    if (currentQRTableNumber) {
        html += `
                <div class="page" style="page-break-after: auto;">
                    <div class="label">Mesa</div>
                    <div class="table-text">${currentQRTableNumber}</div>
                </div>
        `;
    }
    
    html += `
            </body>
        </html>
    `;
    
    printWindow.document.write(html);
    printWindow.document.close();
}

function printAllTableQRs() {
    const btns = document.querySelectorAll('.qr-table-btn');
    if (btns.length === 0) {
        alert("No tables found to print.");
        return;
    }
    
    const printWindow = window.open('', '', 'width=800,height=600');
    let html = `
        <html>
            <head>
                <title>Print All Tables QR</title>
                <style>
                    body { font-family: sans-serif; margin: 0; padding: 0; text-align: center; }
                    .page { page-break-after: always; height: 100vh; display: flex; flex-direction: column; align-items: center; justify-content: center; }
                    .qr-img { width: 80vmin; height: 80vmin; max-width: 800px; max-height: 800px; }
                    .table-text { font-size: 40vmin; font-weight: bold; margin: 0; line-height: 1; }
                    .label { font-size: 6vmin; margin-bottom: 20px; color: #555; text-transform: uppercase; letter-spacing: 2px; }
                    @media print {
                        @page { margin: 0; }
                        body { -webkit-print-color-adjust: exact; }
                    }
                </style>
            </head>
            <body>
    `;
    
    let validUrls = [];
    
    btns.forEach(btn => {
        const tableNumber = btn.dataset.table;
        const tournamentId = btn.dataset.tournamentId;
        const eventId = btn.dataset.eventId;
        
        let scoreUrl;
        if (eventId && eventId !== "" && eventId !== "null" && eventId !== "undefined") {
            scoreUrl = window.location.origin + '/score/e/' + eventId + '/table/' + tableNumber;
        } else if (tournamentId && tournamentId !== "" && tournamentId !== "null" && tournamentId !== "undefined") {
            scoreUrl = window.location.origin + '/score/t/' + tournamentId + '/table/' + tableNumber;
        } else {
            return;
        }
        
        validUrls.push({ tableNumber, scoreUrl });
    });
    
    if (validUrls.length === 0) {
        alert("No valid URLs found to print.");
        printWindow.close();
        return;
    }
    
    validUrls.forEach((item, index) => {
        const src = '/qr?size=1000&data=' + encodeURIComponent(item.scoreUrl);
        
        html += `
                <div class="page">
                    <div class="label">Escanear para Anotar</div>
                    <img class="qr-img" src="${src}" onload="if(++window.loadedCount === ${validUrls.length}) { window.print(); window.close(); }" />
                </div>
                <div class="page" style="${index === validUrls.length - 1 ? 'page-break-after: auto;' : ''}">
                    <div class="label">Mesa</div>
                    <div class="table-text">${item.tableNumber}</div>
                </div>
        `;
    });
    
    html += `
            </body>
            <script>window.loadedCount = 0;</script>
        </html>
    `;
    
    printWindow.document.write(html);
    printWindow.document.close();
}

// ── Toasts & UI Notifications ──────────────────────────────────────────────────
function showToast(message, type = 'success') {
    const container = document.getElementById('toast-container');
    if (!container) return;

    const toast = document.createElement('div');
    const borderClass = type === 'error' ? 'border-red-500/30' : 'border-white/10';
    toast.className = `cursor-pointer flex items-center gap-3 px-5 py-4 rounded-2xl bg-club-panel border ${borderClass} shadow-2xl transition-all duration-500 transform translate-x-full opacity-0 pointer-events-auto max-w-sm`;

    const icon = type === 'error'
        ? `<svg class="w-5 h-5 text-red-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z"></path></svg>`
        : `<svg class="w-5 h-5 text-emerald-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>`;

    toast.innerHTML = `${icon}<span class="text-xs font-bold uppercase tracking-wider text-white/90">${message}</span>`;

    container.appendChild(toast);

    requestAnimationFrame(() => {
        toast.classList.remove('translate-x-full', 'opacity-0');
    });

    const dismiss = () => {
        toast.style.opacity = '0';
        toast.style.transform = 'translateX(100%)';
        setTimeout(() => toast.remove(), 300);
    };
    toast.onclick = dismiss;
    
    // Auto dismiss all toasts, longer for errors
    setTimeout(dismiss, type === 'error' ? 6000 : 4500);
}

// Check for pending success toasts on page load
(function() {
    const pending = sessionStorage.getItem('pending_toast');
    if (pending) {
        showToast(pending, 'success');
        sessionStorage.removeItem('pending_toast');
    }
})();

// Clear pending toast if HTMX returns a response error
document.body.addEventListener('htmx:responseError', function() {
    sessionStorage.removeItem('pending_toast');
});

// ── Global styled confirm (replaces native confirm() for every hx-confirm) ──────
document.body.addEventListener('htmx:confirm', function(evt) {
    if (!evt.detail.question) return; // no hx-confirm on this element, let it proceed

    const modal = document.getElementById('global-confirm-modal');
    const msgEl = document.getElementById('global-confirm-msg');
    const yesBtn = document.getElementById('global-confirm-yes');
    const cancelBtn = document.getElementById('global-confirm-cancel');
    if (!modal || !msgEl || !yesBtn || !cancelBtn) {
        if (window.confirm(evt.detail.question)) evt.detail.issueRequest(true);
        return;
    }

    evt.preventDefault();
    msgEl.textContent = evt.detail.question;
    modal.classList.remove('hidden');

    function cleanup() {
        modal.classList.add('hidden');
        yesBtn.removeEventListener('click', onConfirm);
        cancelBtn.removeEventListener('click', onCancel);
    }
    function onConfirm() {
        cleanup();
        evt.detail.issueRequest(true);
    }
    function onCancel() {
        cleanup();
    }
    yesBtn.addEventListener('click', onConfirm);
    cancelBtn.addEventListener('click', onCancel);
});

// Auto-reload bracket and matches after player move
document.body.addEventListener('htmx:afterOnLoad', function(evt) {
    const path = (evt.detail.pathInfo && evt.detail.pathInfo.requestPath) || 
                 (evt.detail.requestConfig && evt.detail.requestConfig.path) || '';
    if (path.includes('/move-player')) {
        htmx.trigger('#bracket-container', 'reload-bracket');
        htmx.trigger('#custom-matches-list', 'reload-matches');
    }
});

// ── Drag & Drop ── JS reads data-* attributes at drop time → htmx.ajax() POST.
// Server responds with HX-Trigger: reload-bracket → bracket reloads automatically.

function onDragStart(event, playerId) {
    event.dataTransfer.setData('text/plain', playerId);
    event.dataTransfer.effectAllowed = 'move';
    event.currentTarget.classList.add('opacity-40');
}

function onDragEnd(event) {
    event.currentTarget.classList.remove('opacity-40');
}

function onDragOver(event) {
    event.preventDefault();
    event.dataTransfer.dropEffect = 'move';
}

function onDragEnter(event, el) {
    event.preventDefault();
    el.classList.remove('border-white/5');
    el.classList.add('border-red-500/50', 'bg-red-500/5', 'scale-[1.01]');
}

function onDragLeave(event, el) {
    el.classList.add('border-white/5');
    el.classList.remove('border-red-500/50', 'bg-red-500/5', 'scale-[1.01]');
}

function onDrop(event, el) {
    event.preventDefault();
    el.classList.add('border-white/5');
    el.classList.remove('border-red-500/50', 'bg-red-500/5', 'scale-[1.01]');

    const playerId = event.dataTransfer.getData('text/plain');
    if (!playerId) return;

    const tournamentId = el.dataset.tournamentId;
    const targetGroupId = el.dataset.groupId || '';

    htmx.ajax('POST', '/admin/tournaments/' + tournamentId + '/move-player', {
        source: document.body,
        swap: 'none',
        values: { playerId, targetGroupId }
    });
}

function onDragOverRow(event) {
    event.preventDefault();
    event.stopPropagation();
    event.dataTransfer.dropEffect = 'move';
}

function onDragEnterRow(event, el) {
    event.preventDefault();
    event.stopPropagation();
    el.classList.add('border-t-2', 'border-red-500/70', 'bg-red-500/5');
}

function onDragLeaveRow(event, el) {
    event.stopPropagation();
    el.classList.remove('border-t-2', 'border-red-500/70', 'bg-red-500/5');
}

function onDropRow(event, el) {
    event.preventDefault();
    event.stopPropagation();
    el.classList.remove('border-t-2', 'border-red-500/70', 'bg-red-500/5');

    const playerId = event.dataTransfer.getData('text/plain');
    if (!playerId) return;

    const tournamentId = el.dataset.tournamentId;
    const targetGroupId = el.dataset.groupId || '';
    const targetIndex   = parseInt(el.dataset.targetIndex ?? '-1', 10);

    htmx.ajax('POST', '/admin/tournaments/' + tournamentId + '/move-player', {
        source: document.body,
        swap: 'none',
        values: { playerId, targetGroupId, targetIndex }
    });
}

function onDropKnockoutRow(event, el) {
    event.preventDefault();
    event.stopPropagation();
    el.classList.remove('border-t-2', 'border-red-500/70', 'bg-red-500/5');

    const playerId = event.dataTransfer.getData('text/plain');
    if (!playerId) return;

    const tournamentId = el.dataset.tournamentId;
    const divId = el.dataset.divId;
    const targetIndex = parseInt(el.dataset.targetIndex ?? '-1', 10);

    const tbody = el.closest('tbody');
    const rows = Array.from(tbody.querySelectorAll('tr[data-player-id]'));
    let playerIds = rows.map(r => r.dataset.playerId);

    const sourceIndex = playerIds.indexOf(playerId);
    if (sourceIndex > -1 && sourceIndex !== targetIndex) {
        playerIds.splice(sourceIndex, 1);
        playerIds.splice(targetIndex, 0, playerId);
    }

    htmx.ajax('POST', '/admin/tournaments/' + tournamentId + '/save-knockout-seeds', {
        source: document.body,
        swap: 'none',
        values: { divId: divId, playerIds: JSON.stringify(playerIds) }
    });
}

var currentQRTableNumber = '';

function showQRCodeModal(matchId, matchup, tableNumber, tournamentId, eventId) {
    const modal = document.getElementById('qr-modal');
    const matchupEl = document.getElementById('qr-matchup');
    const imageEl = document.getElementById('qr-image');
    const copyBtn = document.getElementById('qr-copy-btn');
    const openBtn = document.getElementById('qr-open-btn');
    
    currentQRTableNumber = tableNumber || '';
    
    let scoreUrl;
    if (tableNumber && tableNumber !== "" && tableNumber !== "null" && tableNumber !== "undefined") {
        if (eventId && eventId !== "" && eventId !== "null" && eventId !== "undefined") {
            scoreUrl = window.location.origin + '/score/e/' + eventId + '/table/' + tableNumber;
        } else if (tournamentId && tournamentId !== "" && tournamentId !== "null" && tournamentId !== "undefined") {
            scoreUrl = window.location.origin + '/score/t/' + tournamentId + '/table/' + tableNumber;
        } else {
            scoreUrl = window.location.origin + '/score/' + matchId;
        }
    } else {
        scoreUrl = window.location.origin + '/score/' + matchId;
    }
    
    matchupEl.textContent = matchup + (tableNumber ? ' (Table ' + tableNumber + ')' : '');
    imageEl.src = '/qr?size=1000&data=' + encodeURIComponent(scoreUrl);
    openBtn.href = scoreUrl;
    
    copyBtn.onclick = function() {
        navigator.clipboard.writeText(scoreUrl).then(() => {
            showToast('📋 Link copied to clipboard!', 'success');
        }).catch(err => {
            console.error('Could not copy text: ', err);
        });
    };
    
    modal.classList.remove('hidden');
}

function openScoreModal(url) {
    document.getElementById('score-modal').classList.remove('hidden');
    htmx.ajax('GET', url, { target: '#score-modal-body', swap: 'innerHTML' });
}

document.addEventListener('DOMContentLoaded', () => {
  const initDragToScroll = () => {
    const containers = document.querySelectorAll('.overflow-x-auto');
    containers.forEach(ele => {
      if (ele.dataset.dragInitialized) return;
      ele.dataset.dragInitialized = 'true';
      
      ele.style.cursor = 'grab';
      let pos = { top: 0, left: 0, x: 0, y: 0 };
      
      const mouseDownHandler = function(e) {
        if (e.target.closest('[draggable="true"]') || e.target.closest('button') || e.target.closest('a')) {
          return;
        }
        e.preventDefault(); // Prevent Firefox from starting native drag which gets the page stuck
        ele.style.cursor = 'grabbing';
        ele.style.userSelect = 'none';
        pos = {
          left: ele.scrollLeft,
          top: ele.scrollTop,
          x: e.clientX,
          y: e.clientY,
        };

        document.addEventListener('mousemove', mouseMoveHandler);
        document.addEventListener('mouseup', mouseUpHandler);
      };

      const mouseMoveHandler = function(e) {
        const dx = e.clientX - pos.x;
        const dy = e.clientY - pos.y;
        ele.scrollTop = pos.top - dy;
        ele.scrollLeft = pos.left - dx;
      };

      const mouseUpHandler = function() {
        document.removeEventListener('mousemove', mouseMoveHandler);
        document.removeEventListener('mouseup', mouseUpHandler);
        ele.style.cursor = 'grab';
        ele.style.removeProperty('user-select');
      };

      ele.addEventListener('mousedown', mouseDownHandler);
    });
  };
  
  initDragToScroll();
  document.body.addEventListener('htmx:afterSwap', () => {
      initDragToScroll();
  });
});
