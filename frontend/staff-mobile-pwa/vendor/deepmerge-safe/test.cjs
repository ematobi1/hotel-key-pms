const assert = require('node:assert');
const merge = require('./index.js');
// behaviour parity
assert.deepStrictEqual(merge({ a: 1, n: { x: 1 } }, { b: 2, n: { y: 2 } }), { a: 1, b: 2, n: { x: 1, y: 2 } });
assert.deepStrictEqual(merge([1], [2]), [1, 2]);
assert.deepStrictEqual(merge.all([{ a: 1 }, { b: 2 }, { c: 3 }]), { a: 1, b: 2, c: 3 });
const src = { deep: { v: 1 } }; const out = merge({}, src); out.deep.v = 9; assert.strictEqual(src.deep.v, 1); // clones
// attack: prototype poisoning via JSON-parsed __proto__ / constructor
const evil = JSON.parse('{"__proto__":{"polluted":true},"constructor":{"prototype":{"polluted":true}},"ok":1}');
const res = merge({}, evil);
assert.strictEqual(res.polluted, undefined);
assert.strictEqual(Object.getPrototypeOf(res), Object.prototype);
assert.strictEqual(({}).polluted, undefined);
assert.strictEqual(res.ok, 1);
console.log('deepmerge-safe: all assertions passed');
