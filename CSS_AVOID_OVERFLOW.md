# CSS Overflow Prevention Guide

A systematic guide to preventing child elements from expanding beyond the viewport.

## The Root Problem

By default, CSS flex and grid items have `min-width: auto`, meaning they **won't shrink smaller than their content**. This causes unexpected horizontal overflow when content is wider than the viewport.

```css
/* This WON'T work as expected */
.container {
  display: flex;
  max-width: 100%;
}

.content {
  /* This will overflow if content is wider than container */
}
```

## The Golden Rule: Add `min-width: 0` to All Flex/Grid Containers

Every element in the chain from viewport to content needs `min-width: 0`:

```css
/* Layout containers */
.page { 
  display: flex; 
  min-width: 0;  /* Critical! */ 
}

.sidebar { 
  display: flex; 
  min-width: 0;  /* Critical! */ 
}

.content { 
  display: flex; 
  min-width: 0;  /* Critical! */ 
}

/* Text containers */
.card { 
  min-width: 0;  /* Critical! */ 
  overflow: hidden;  /* Optional extra safety */
}

.text { 
  word-break: break-all;  /* Or break-word */ 
}
```

## Complete Solution Pattern

```css
/* Root container */
.app {
  display: flex;
  flex-direction: column;
  max-width: 100vw;
  min-width: 0;
  overflow-x: hidden;
}

/* Page wrapper */
.page {
  display: flex;
  flex-direction: column;
  max-width: 100%;
  min-width: 0;
  flex: 1;
}

/* Section containers */
.section {
  padding: 0 16px;
  min-width: 0;  /* Don't forget this! */
}

/* Card/component containers */
.card {
  display: flex;
  flex-direction: column;
  min-width: 0;  /* Critical for nested flex */
  overflow: hidden;
}

/* Content that might overflow */
.code-block,
.token-display,
.file-path {
  word-break: break-all;
  overflow-wrap: break-word;
  min-width: 0;
}
```

## Text Wrapping Strategies

### 1. Break All (for code, paths, tokens)
```css
.break-all {
  word-break: break-all;
}
/* hello/world/very/long/path/file.txt */
```

### 2. Break Word (for readable text)
```css
.break-word {
  overflow-wrap: break-word;
  word-wrap: break-word;
}
/* This is a verylongwordthatneedsbreaking */
```

### 3. Ellipsis (for single-line truncation)
```css
.ellipsis {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;  /* Required for flex children */
}
```

### 4. Anywhere (modern browsers)
```css
.break-anywhere {
  overflow-wrap: anywhere;
}
```

## Grid-Specific Solution

Grid has the same problem with `min-width: auto`:

```css
.grid-container {
  display: grid;
  grid-template-columns: 200px 1fr;
  min-width: 0;  /* Required! */
}

.grid-item {
  min-width: 0;  /* Required for grid items! */
  overflow: hidden;
}
```

## Global CSS Reset (Nuclear Option)

Apply `min-width: 0` to everything by default:

```css
/* Reset all flex/grid items */
* {
  min-width: 0;
  min-height: 0;
}

/* Then explicitly set where you WANT minimums */
button,
input,
select {
  min-width: auto;
}
```

## Common Scenarios

### Scenario 1: Long URLs/File Paths
```css
.file-path {
  display: block;
  min-width: 0;
  word-break: break-all;
  font-family: monospace;
}
```

### Scenario 2: Tables with Long Content
```css
.table-container {
  max-width: 100%;
  overflow-x: auto;
}

table {
  width: 100%;
  table-layout: fixed;
}

td {
  word-break: break-all;
  min-width: 0;
}
```

### Scenario 3: Cards with Code Blocks
```css
.card {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.card-content {
  min-width: 0;
  overflow: hidden;
}

pre {
  overflow-x: auto;
  max-width: 100%;
  word-break: break-all;
}
```

### Scenario 4: Sidebar + Main Layout
```css
.layout {
  display: flex;
  min-width: 0;
}

.sidebar {
  width: 250px;
  flex-shrink: 0;
}

.main {
  flex: 1;
  min-width: 0;  /* Critical! */
  overflow: hidden;
}
```

## Debugging Tips

### 1. Visual Debugging
```css
/* Add this temporarily to find overflow */
* {
  outline: 1px solid red;
}
```

### 2. Find the Overflowing Element
```javascript
// Run in console to find overflowing elements
document.querySelectorAll('*').forEach(el => {
  if (el.scrollWidth > el.clientWidth) {
    console.log('Overflowing:', el);
    el.style.outline = '2px solid red';
  }
});
```

### 3. Check Computed Styles
Look for elements with:
- `display: flex` or `display: grid`
- Missing `min-width: 0`
- `white-space: nowrap` on long content

## Quick Checklist

When building a new component:

- [ ] Does any ancestor use `display: flex` or `display: grid`?
- [ ] If yes, did you add `min-width: 0` to all flex/grid containers?
- [ ] Does the content include long strings (paths, URLs, tokens)?
- [ ] If yes, did you add `word-break: break-all` or `overflow-wrap: break-word`?
- [ ] Did you test with very long content (100+ characters)?
- [ ] Is `overflow-x: hidden` set on the root container?

## Why This Happens

The CSS spec defines:
- `min-width: auto` = minimum of content width
- Flex/grid items can't shrink below `min-width`
- This overrides `max-width: 100%` and `width: 100%`

The solution is explicitly setting `min-width: 0` to allow shrinking below content width.

## Browser Support

- `min-width: 0` - All modern browsers ✅
- `overflow-wrap: anywhere` - Modern browsers (not IE) ⚠️
- `word-break: break-word` - Deprecated, use `overflow-wrap` instead ⚠️

## References

- [CSS Flexible Box Layout Module](https://www.w3.org/TR/css-flexbox-1/)
- [CSS Grid Layout Module](https://www.w3.org/TR/css-grid-1/)
- [MDN: min-width](https://developer.mozilla.org/en-US/docs/Web/CSS/min-width)
