'use strict';
// Hardened drop-in replacement for deepmerge@4.x (MIT, Josh Duff and contributors).
// Same API: deepmerge(target, source, options) and deepmerge.all(array, options).
// Fix for GHSA-gcqx-w48m-w2h4: prototype-poisoning keys are never read or written,
// and merged objects are created with the same prototype as the source.
const FORBIDDEN = new Set(['__proto__', 'constructor', 'prototype']);

function isMergeableObject(value) {
  if (!value || typeof value !== 'object') return false;
  const tag = Object.prototype.toString.call(value);
  return tag !== '[object RegExp]' && tag !== '[object Date]';
}

function emptyTarget(val) {
  return Array.isArray(val) ? [] : {};
}

function cloneUnlessOtherwiseSpecified(value, options) {
  return options.clone !== false && options.isMergeableObject(value)
    ? deepmerge(emptyTarget(value), value, options)
    : value;
}

function defaultArrayMerge(target, source, options) {
  return target.concat(source).map((el) => cloneUnlessOtherwiseSpecified(el, options));
}

function getMergeFunction(key, options) {
  if (!options.customMerge) return deepmerge;
  const custom = options.customMerge(key);
  return typeof custom === 'function' ? custom : deepmerge;
}

function getKeys(target) {
  const symbols = Object.getOwnPropertySymbols
    ? Object.getOwnPropertySymbols(target).filter((s) => Object.prototype.propertyIsEnumerable.call(target, s))
    : [];
  return Object.keys(target).concat(symbols).filter((k) => !FORBIDDEN.has(k));
}

function propertyIsUnsafe(target, key) {
  if (FORBIDDEN.has(key)) return true;
  try {
    return key in target
      && !(Object.prototype.hasOwnProperty.call(target, key)
        && Object.prototype.propertyIsEnumerable.call(target, key));
  } catch {
    return false;
  }
}

function mergeObject(target, source, options) {
  const destination = {};
  if (options.isMergeableObject(target)) {
    for (const key of getKeys(target)) {
      destination[key] = cloneUnlessOtherwiseSpecified(target[key], options);
    }
  }
  for (const key of getKeys(source)) {
    if (propertyIsUnsafe(target, key)) continue;
    if (Object.prototype.hasOwnProperty.call(target, key) && options.isMergeableObject(source[key])) {
      destination[key] = getMergeFunction(key, options)(target[key], source[key], options);
    } else {
      destination[key] = cloneUnlessOtherwiseSpecified(source[key], options);
    }
  }
  return destination;
}

function deepmerge(target, source, options) {
  options = options || {};
  options.arrayMerge = options.arrayMerge || defaultArrayMerge;
  options.isMergeableObject = options.isMergeableObject || isMergeableObject;
  options.cloneUnlessOtherwiseSpecified = cloneUnlessOtherwiseSpecified;

  const sourceIsArray = Array.isArray(source);
  const targetIsArray = Array.isArray(target);
  if (sourceIsArray !== targetIsArray) return cloneUnlessOtherwiseSpecified(source, options);
  if (sourceIsArray) return options.arrayMerge(target, source, options);
  return mergeObject(target, source, options);
}

deepmerge.all = function deepmergeAll(array, options) {
  if (!Array.isArray(array)) throw new Error('first argument should be an array');
  return array.reduce((prev, next) => deepmerge(prev, next, options), {});
};

module.exports = deepmerge;
module.exports.default = deepmerge;
