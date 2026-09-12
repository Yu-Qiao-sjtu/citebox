'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const modulePath = path.resolve(__dirname, '..', 'ai-conversation-view.js');

function loadView(options = {}) {
    const code = fs.readFileSync(modulePath, 'utf8');
    const context = {
        console: console,
        Blob: Blob,
        fetch: options.fetch || (() => { throw new Error('fetch not stubbed'); }),
        navigator: options.navigator || {},
        document: {
            dispatchEvent() {},
            addEventListener() {},
            createElement(tag) {
                return {
                    tag,
                    value: '',
                    style: {},
                    removed: false,
                    focused: false,
                    selected: false,
                    focus() { this.focused = true; },
                    select() { this.selected = true; },
                    remove() { this.removed = true; },
                };
            },
            lastTextarea: null,
            body: {
                appendChild(el) { this.owner.lastTextarea = el; },
                owner: null,
            },
            execCommand() {
                return options.execCommandResult !== undefined ? options.execCommandResult : true;
            },
        },
        window: {
            isSecureContext: options.isSecureContext !== undefined ? options.isSecureContext : false,
            AIReader: {
                toolTags: {
                    parseToolTags: () => ({ intentHint: '', sources: [], conflict: null }),
                    renderMentionHTML: (value) => value,
                },
            },
        },
        localStorage: {
            getItem() { return null; },
            setItem() {},
        },
        Event: class Event {
            constructor(type, init) {
                this.type = type;
                this.bubbles = !!(init && init.bubbles);
            }
        },
        CustomEvent: class CustomEvent {
            constructor(type, init) {
                this.type = type;
                this.detail = init && init.detail;
            }
        },
        t(key, fallback) {
            return fallback || key;
        },
    };
    context.document.body.owner = context.document;
    context.globalThis = context;

    vm.createContext(context);
    context.__utils = options.utils || {};
    vm.runInContext('const Utils = __utils;', context);
    vm.runInContext(code, context, { filename: modulePath });
    return { view: context.window.AIReader.view, context };
}

function createSubject(view, els, extraState = {}) {
    const subject = Object.create(view);
    subject._state = {
        els,
        conversationId: 5,
        pinnedPapers: [],
        messages: [],
        turnRuns: [],
        pendingCitations: [],
        rewriteLast: false,
        rewriteOriginalMessageID: null,
        streaming: null,
        ...extraState,
    };
    return subject;
}

function buildEls() {
    return {
        exportBtn: { disabled: false, textContent: '导出' },
        exportModal: {
            classList: {
                _hidden: true,
                add(cls) { if (cls === 'hidden') this._hidden = true; },
                remove(cls) { if (cls === 'hidden') this._hidden = false; },
                contains(cls) { return cls === 'hidden' && this._hidden !== false; },
            },
        },
        exportModalBody: { textContent: '', scrollTop: 100, style: {} },
    };
}

test('_openExportModal renders export in app without navigation', async () => {
    let fetchUrl = '';
    const { view } = loadView({
        fetch: async (url) => {
            fetchUrl = url;
            return {
                ok: true,
                headers: { get: (name) => (name === 'Content-Disposition' ? 'attachment; filename="conv-5.md"' : null) },
                text: async () => '# 对话内容\n\n正文',
            };
        },
    });
    const els = buildEls();
    const subject = createSubject(view, els);

    await subject._openExportModal();

    assert.equal(fetchUrl, '/api/ai/conversations/5/export');
    assert.equal(els.exportModalBody.textContent, '# 对话内容\n\n正文');
    assert.equal(els.exportModal.classList._hidden, false);
    assert.equal(subject._state.exportFilename, 'conv-5.md');
    assert.equal(els.exportBtn.disabled, false);
    assert.equal(els.exportBtn.textContent, '导出');
});

test('_openExportModal falls back to a generated filename without disposition header', async () => {
    const { view } = loadView({
        fetch: async () => ({
            ok: true,
            headers: { get: () => null },
            text: async () => 'content',
        }),
    });
    const els = buildEls();
    const subject = createSubject(view, els);

    await subject._openExportModal();

    assert.equal(subject._state.exportFilename, 'citebox-conversation-5.md');
});

test('_openExportModal keeps the window in place when the export request fails', async () => {
    const toasts = [];
    const { view } = loadView({
        fetch: async () => ({ ok: false, headers: { get: () => null } }),
        utils: {
            showToast(message, type) { toasts.push({ message, type }); },
        },
    });
    const els = buildEls();
    const subject = createSubject(view, els);

    await subject._openExportModal();

    assert.equal(els.exportModal.classList._hidden, true);
    assert.equal(els.exportBtn.disabled, false);
    assert.equal(toasts.length, 1);
});

test('_closeExportModal hides the modal again', async () => {
    const { view } = loadView({});
    const els = buildEls();
    const subject = createSubject(view, els, { exportMarkdown: 'text' });
    els.exportModal.classList._hidden = false;

    subject._closeExportModal();

    assert.equal(els.exportModal.classList._hidden, true);
});

test('_copyExport prefers the async clipboard API', async () => {
    const written = [];
    const toasts = [];
    const { view } = loadView({
        isSecureContext: true,
        navigator: {
            clipboard: {
                writeText: async (text) => { written.push(text); },
            },
        },
        utils: {
            showToast(message, type) { toasts.push({ message, type }); },
        },
    });
    const els = buildEls();
    const subject = createSubject(view, els, { exportMarkdown: '全文内容' });

    await subject._copyExport();

    assert.deepEqual(written, ['全文内容']);
    assert.equal(toasts[0].type, 'success');
});

test('_copyExport falls back to execCommand when the clipboard API is unavailable', async () => {
    const toasts = [];
    const { view, context } = loadView({
        utils: {
            showToast(message, type) { toasts.push({ message, type }); },
        },
    });
    const els = buildEls();
    const subject = createSubject(view, els, { exportMarkdown: '全文内容' });

    await subject._copyExport();

    const ta = context.document.lastTextarea;
    assert.equal(toasts[0].type, 'success');
    assert.ok(ta, 'fallback textarea created');
    assert.equal(ta.selected, true);
    assert.equal(ta.removed, true);
});

test('_downloadExport delegates to Utils.saveBlobDownload with the export filename', async () => {
    const calls = [];
    const { view } = loadView({
        utils: {
            saveBlobDownload: async (blob, filename) => {
                calls.push({ size: blob.size, filename });
                return true;
            },
        },
    });
    const els = buildEls();
    const subject = createSubject(view, els, { exportMarkdown: '# hello', exportFilename: 'conv-5.md' });

    await subject._downloadExport();

    assert.equal(calls.length, 1);
    assert.equal(calls[0].filename, 'conv-5.md');
    assert.equal(calls[0].size, 7);
});
