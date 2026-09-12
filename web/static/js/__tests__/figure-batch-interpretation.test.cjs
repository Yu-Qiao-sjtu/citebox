'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

// Model bubbling through the real modal hierarchy, including the sibling
// figure and batch dialogs. A handler on figureModalBody must not see batch clicks.
class Element {
    constructor(parent = null, dataset = {}) {
        this.parent = parent;
        this.dataset = dataset;
        this.listeners = {};
        this.classList = { add() {}, remove() {}, contains() { return false; } };
    }
    addEventListener(type, handler) { (this.listeners[type] ||= []).push(handler); }
    querySelector() { return null; }
    closest(selector) {
        if (selector === '[data-batch-interpretation-action]' && this.dataset.batchInterpretationAction) return this;
        if (selector === '[data-batch-interpretation-option]' && this.dataset.batchInterpretationOption) return this;
        return null;
    }
    async dispatch(type) {
        for (let node = this; node; node = node.parent) {
            for (const handler of node.listeners[type] || []) await handler({ target: this });
        }
    }
}
function setup(api = {}) {
    const root = new Element();
    const nodes = {};
    for (const prefix of ['figure', 'figureInterpretation', 'figureBatchInterpretation']) {
        nodes[`${prefix}Modal`] = new Element(root);
        nodes[`${prefix}ModalBody`] = new Element(nodes[`${prefix}Modal`]);
        const close = `close${prefix[0].toUpperCase()}${prefix.slice(1)}Modal`;
        nodes[close] = new Element(nodes[`${prefix}Modal`]);
    }
    const context = {
        window: {}, document: { getElementById: id => nodes[id], body: root, addEventListener() {} },
        AbortController, HTMLElement: Element, console,
        t: (key, fallback) => fallback || key,
        Utils: { showToast() {}, escapeHTML: value => String(value) }, API: api
    };
    vm.runInNewContext(fs.readFileSync(path.join(__dirname, '..', 'figure-viewer.js'), 'utf8') + '\nglobalThis.viewer = FigureViewer;', context);
    const viewer = context.viewer;
    viewer.init();
    viewer.currentFigure = { id: 1, paper_id: 11 };
    const figures = [
        { id: 1, figure_index: 1, notes_text: '' },
        { id: 2, figure_index: 2, notes_text: 'Human note' },
        { id: 3, figure_index: 3, notes_text: '' },
        { id: 4, figure_index: 3, parent_figure_id: 3, notes_text: '' }
    ];
    viewer.paperDetails.set(11, { id: 11, figures });
    viewer.render = () => {};
    viewer.openBatchInterpretationModal();
    const action = name => new Element(viewer.batchBody, { batchInterpretationAction: name });
    const option = (name, value) => Object.assign(new Element(viewer.batchBody, { batchInterpretationOption: name }), { value });
    return { viewer, figures, action, option };
}

test('batch sibling dialog starts serially, skips existing notes and continues after failure', async () => {
    const calls = [];
    const saves = [];
    const { viewer, action } = setup({
        async readPaperWithAI(request) {
            calls.push(request.figure_id);
            if (request.figure_id === 1) throw new Error('Model unavailable');
            return { answer: 'Interpretation' };
        },
        async updateFigure(id, data) { saves.push({ id, ...data }); return {}; }
    });
    await action('start').dispatch('click');
    assert.deepEqual(calls, [1, 3]);
    assert.deepEqual(saves, [{ id: 3, notes_text: 'Interpretation' }]);
    assert.equal(viewer.batchState.succeeded, 1);
    assert.equal(viewer.batchState.skipped, 1);
    assert.equal(viewer.batchState.failures.length, 1);
    assert.equal(viewer.batchState.done, true);
});

test('batch scope and note mode respond to change events', async () => {
    const saves = [];
    const { viewer, action, option } = setup({
        async readPaperWithAI() { return { answer: 'New answer' }; },
        async updateFigure(id, data) { saves.push({ id, ...data }); return {}; }
    });
    await option('scope', 'all').dispatch('change');
    await option('mode', 'overwrite').dispatch('change');
    assert.equal(viewer.batchState.options.scope, 'all');
    await action('start').dispatch('click');
    assert.equal(saves.length, 3);
    assert.equal(saves.find(save => save.id === 2).notes_text, 'New answer');
});

test('append keeps existing notes when all figures are selected', async () => {
    const saves = [];
    const { action, option } = setup({
        async readPaperWithAI() { return { answer: 'New answer' }; },
        async updateFigure(id, data) { saves.push({ id, ...data }); return {}; }
    });
    await option('scope', 'all').dispatch('change');
    await action('start').dispatch('click');
    assert.equal(saves.find(save => save.id === 2).notes_text, 'Human note\n\nNew answer');
});

test('interrupt cancels pending inference, prevents duplicate starts and does not save its answer', async () => {
    let started;
    const waiting = new Promise(resolve => { started = resolve; });
    let calls = 0;
    const { viewer, action } = setup({
        readPaperWithAI(request, { signal }) {
            calls++;
            started();
            return new Promise((resolve, reject) => signal.addEventListener('abort', () => reject(Object.assign(new Error('Aborted'), { name: 'AbortError' }))));
        },
        async updateFigure() { assert.fail('Cancelled inference must not save'); }
    });
    const run = action('start').dispatch('click');
    await waiting;
    await action('start').dispatch('click');
    await action('stop').dispatch('click');
    await run;
    assert.equal(calls, 1);
    assert.equal(viewer.batchState.abort, true);
    assert.equal(viewer.batchState.running, false);
    assert.equal(viewer.batchState.failures.length, 0);
    assert.match(viewer.batchBody.innerHTML, /尚有 2 张未处理/);
});
