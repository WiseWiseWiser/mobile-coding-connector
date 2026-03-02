# iOS Safari Input Zooming Fix

## Problem
On iOS Safari (iPhone), focusing on input fields causes the browser to zoom in automatically. This is an accessibility feature triggered when:
- Input font-size is less than 16px
- The viewport allows scaling

## Solution Applied

### 1. Viewport Meta Tag (`index.html:7`)

```html
<meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no, viewport-fit=cover" />
```

**Key attributes:**
- `maximum-scale=1.0` - Prevents zooming beyond initial scale
- `user-scalable=no` - Disables user zooming entirely
- `viewport-fit=cover` - Ensures content extends to device edges on notched displays

### 2. Global CSS (`index.css`)

#### Method 1: Target Touch Devices
```css
@media (hover: none) and (pointer: coarse) {
  input,
  select,
  textarea,
  [contenteditable="true"] {
    font-size: 16px !important;
  }
}
```
- Uses CSS 4 Media Queries Level 4
- Targets devices that don't support hover (touchscreens)
- Applies to coarse pointer devices (finger touch)

#### Method 2: Target iOS Specifically
```css
@supports (-webkit-touch-callout: none) {
  input, select, textarea {
    font-size: 16px !important;
  }
}
```
- Uses CSS Feature Queries
- `-webkit-touch-callout` is iOS Safari specific
- Acts as a browser detection mechanism

#### Method 3: Prevent Double-Tap Zoom
```css
input, select, textarea, button {
  touch-action: manipulation;
  -webkit-tap-highlight-color: transparent;
}
```
- `touch-action: manipulation` - Disables browser handling of gestures
- `-webkit-tap-highlight-color` - Removes tap highlight on mobile browsers

## Files Modified

1. **`ai-critic-react/index.html`**
   - Updated viewport meta tag with zoom prevention attributes

2. **`ai-critic-react/src/index.css`**
   - Added global CSS rules to prevent iOS zooming
   - Three different targeting methods for maximum compatibility

## Testing

To verify the fix works:
1. Open the application on an iOS device (iPhone/iPad)
2. Navigate to any form with inputs (e.g., TODO list, ACP Chat)
3. Tap on an input field
4. **Expected behavior:** The page should NOT zoom in

## Browser Compatibility

- **iOS Safari:** ✅ Primary target - all methods work
- **iOS Chrome:** ✅ Inherits iOS WebKit behavior
- **Android Chrome:** ✅ CSS media queries apply
- **Desktop browsers:** ✅ No adverse effects (queries don't match)

## References

- [Apple Developer: Supported Meta Tags](https://developer.apple.com/library/archive/documentation/AppleApplications/Reference/SafariHTMLRef/Articles/MetaTags.html)
- [MDN: touch-action CSS Property](https://developer.mozilla.org/en-US/docs/Web/CSS/touch-action)
- [CSS Tricks: Hover Media Query](https://css-tricks.com/interaction-media-features-and-their-potential-for-incorrect-assumptions/)
