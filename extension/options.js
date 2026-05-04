// options.js — Extension options page logic.
// Reads and writes {elementId, urlPattern} tuples via chrome.storage.sync (D-01, D-03).

import { DEFAULT_TARGETS } from './config.js';

// ---------------------------------------------------------------------------
// Storage helpers
// ---------------------------------------------------------------------------

async function getTargets() {
  const result = await chrome.storage.sync.get({ targets: DEFAULT_TARGETS });
  return result.targets;
}

async function saveTargets(targets) {
  await chrome.storage.sync.set({ targets });
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

/**
 * Validate a single tuple. Returns an array of field-level errors.
 * @param {{ elementId: string, urlPattern: string }} tuple
 * @returns {{ elementIdError: string|null, urlPatternError: string|null }}
 */
function validateTuple(tuple) {
  const result = { elementIdError: null, urlPatternError: null };

  if (!tuple.elementId.trim()) {
    result.elementIdError = 'Required';
  }

  if (!tuple.urlPattern.trim()) {
    result.urlPatternError = 'Required';
  } else if (!/^(https?:\/\/|file:\/\/)/.test(tuple.urlPattern.trim())) {
    result.urlPatternError = 'Must start with http://, https://, or file://';
  }

  return result;
}

// ---------------------------------------------------------------------------
// DOM rendering
// ---------------------------------------------------------------------------

let tuples = [];

function renderTupleList() {
  const list = document.getElementById('tuple-list');
  const empty = document.getElementById('empty-message');

  list.innerHTML = ''; // Clear and rebuild — simple state, low row count.

  if (tuples.length === 0) {
    empty.style.display = '';
    return;
  }
  empty.style.display = 'none';

  tuples.forEach((tuple, index) => {
    const row = document.createElement('div');
    row.className = 'tuple-row';
    row.dataset.index = index;

    // Element ID field
    const idField = document.createElement('div');
    idField.className = 'tuple-field';

    const idLabel = document.createElement('label');
    idLabel.setAttribute('for', `elementId-${index}`);
    idLabel.textContent = 'Element ID';

    const idInput = document.createElement('input');
    idInput.type = 'text';
    idInput.id = `elementId-${index}`;
    idInput.name = `elementId-${index}`;
    idInput.placeholder = 'e.g. draw-canvas';
    idInput.value = tuple.elementId;
    idInput.addEventListener('input', () => {
      tuples[index] = { ...tuples[index], elementId: idInput.value };
    });

    const idError = document.createElement('span');
    idError.className = 'field-error';
    idError.id = `elementId-error-${index}`;

    idField.appendChild(idLabel);
    idField.appendChild(idInput);
    idField.appendChild(idError);

    // URL Pattern field
    const urlField = document.createElement('div');
    urlField.className = 'tuple-field';

    const urlLabel = document.createElement('label');
    urlLabel.setAttribute('for', `urlPattern-${index}`);
    urlLabel.textContent = 'URL Pattern';

    const urlInput = document.createElement('input');
    urlInput.type = 'text';
    urlInput.id = `urlPattern-${index}`;
    urlInput.name = `urlPattern-${index}`;
    urlInput.placeholder = 'e.g. file:///C:/App/*.html';
    urlInput.value = tuple.urlPattern;
    urlInput.addEventListener('input', () => {
      tuples[index] = { ...tuples[index], urlPattern: urlInput.value };
    });

    const urlError = document.createElement('span');
    urlError.className = 'field-error';
    urlError.id = `urlPattern-error-${index}`;

    urlField.appendChild(urlLabel);
    urlField.appendChild(urlInput);
    urlField.appendChild(urlError);

    // Remove button
    const removeBtn = document.createElement('button');
    removeBtn.className = 'btn-destructive';
    removeBtn.textContent = 'Remove target';
    removeBtn.type = 'button';
    removeBtn.addEventListener('click', () => {
      tuples.splice(index, 1);
      renderTupleList(); // Re-render — no undo (UI-SPEC.md Copywriting Contract).
    });

    row.appendChild(idField);
    row.appendChild(urlField);
    row.appendChild(removeBtn);
    list.appendChild(row);
  });
}

// ---------------------------------------------------------------------------
// Event handlers
// ---------------------------------------------------------------------------

document.getElementById('btn-add').addEventListener('click', () => {
  tuples.push({ elementId: '', urlPattern: '' });
  renderTupleList();

  // Focus management: move focus to the new row's first input (UI-SPEC.md Accessibility Contract).
  const newIndex = tuples.length - 1;
  const firstInput = document.getElementById(`elementId-${newIndex}`);
  if (firstInput) firstInput.focus();
});

document.getElementById('btn-save').addEventListener('click', async () => {
  // Validate all tuples before saving.
  let valid = true;

  tuples.forEach((tuple, index) => {
    const errors = validateTuple(tuple);
    const idInput  = document.getElementById(`elementId-${index}`);
    const urlInput = document.getElementById(`urlPattern-${index}`);
    const idError  = document.getElementById(`elementId-error-${index}`);
    const urlError = document.getElementById(`urlPattern-error-${index}`);

    if (errors.elementIdError) {
      idInput.classList.add('invalid');
      idError.textContent = errors.elementIdError;
      valid = false;
    } else {
      idInput.classList.remove('invalid');
      idError.textContent = '';
    }

    if (errors.urlPatternError) {
      urlInput.classList.add('invalid');
      urlError.textContent = errors.urlPatternError;
      valid = false;
    } else {
      urlInput.classList.remove('invalid');
      urlError.textContent = '';
    }
  });

  if (!valid) return;

  const status = document.getElementById('status-message');

  try {
    await saveTargets(tuples);
    status.textContent = 'Saved.';
    setTimeout(() => { status.textContent = ''; }, 2000);
  } catch (err) {
    status.textContent = 'Save failed — storage quota exceeded.';
  }
});

// ---------------------------------------------------------------------------
// Bootstrap
// ---------------------------------------------------------------------------

(async () => {
  tuples = await getTargets();
  renderTupleList();
})();
