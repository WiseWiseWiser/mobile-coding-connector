#!/usr/bin/env node
/**
 * Quick Test Skill - Puppeteer-based browser testing for OpenCode Settings
 * 
 * This skill tests the OpenCode settings page to verify port configuration works correctly.
 * 
 * Usage:
 *   go run ./script/run quick-test
 *   Then in another terminal:
 *   node /root/mobile-coding-connector/skills/quick-test-server-and-frontend/test-opencode-settings.js
 * 
 * Or with custom URL:
 *   TEST_URL=http://localhost:37651 node test-opencode-settings.js
 */

const puppeteer = require('puppeteer');
const fs = require('fs');
const path = require('path');

// Configuration
const BASE_URL = process.env.TEST_URL || 'http://localhost:37651';
const SETTINGS_PATH = '/project/mobile-coding-connector/agent/opencode/settings';
const SCREENSHOT_DIR = '/tmp/opencode-test-screenshots';
const TEST_TIMEOUT = 30000; // 30 seconds

// Ensure screenshot directory exists
if (!fs.existsSync(SCREENSHOT_DIR)) {
  fs.mkdirSync(SCREENSHOT_DIR, { recursive: true });
}

// Test results
const results = {
  success: false,
  tests: [],
  errors: [],
  screenshots: []
};

function log(message) {
  const timestamp = new Date().toISOString();
  console.log(`[${timestamp}] ${message}`);
}

async function delay(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function takeScreenshot(page, name) {
  const timestamp = Date.now();
  const filename = `${SCREENSHOT_DIR}/${name}-${timestamp}.png`;
  try {
    await page.screenshot({ path: filename, fullPage: true });
    results.screenshots.push(filename);
    log(`📸 Screenshot saved: ${filename}`);
    return filename;
  } catch (error) {
    log(`⚠️ Failed to take screenshot: ${error.message}`);
    return null;
  }
}

async function testPageLoad(page) {
  log('📍 Test 1: Page Load');
  try {
    await page.goto(`${BASE_URL}${SETTINGS_PATH}`, { 
      waitUntil: 'networkidle2',
      timeout: TEST_TIMEOUT
    });
    await delay(2000);
    await takeScreenshot(page, '01-page-loaded');
    
    const title = await page.title();
    log(`   Page title: ${title}`);
    
    results.tests.push({ name: 'Page Load', status: 'passed' });
    return true;
  } catch (error) {
    log(`   ❌ Failed: ${error.message}`);
    results.tests.push({ name: 'Page Load', status: 'failed', error: error.message });
    results.errors.push({ test: 'Page Load', error: error.message });
    return false;
  }
}

async function testPortFieldExists(page) {
  log('📍 Test 2: Port Field Exists');
  try {
    // Try to find the port input field
    // Based on the code, it's a number input
    const portInput = await page.$('input[type="number"]');
    
    if (portInput) {
      const portValue = await page.evaluate(el => el.value, portInput);
      log(`   ✓ Port input found, value: ${portValue}`);
      await takeScreenshot(page, '02-port-field-found');
      results.tests.push({ name: 'Port Field Exists', status: 'passed', value: portValue });
      return { exists: true, value: portValue, element: portInput };
    } else {
      // Try to find all inputs to help debug
      const allInputs = await page.$$('input');
      log(`   ⚠ Port input not found. Found ${allInputs.length} total inputs:`);
      
      for (let i = 0; i < Math.min(allInputs.length, 10); i++) {
        const type = await page.evaluate(el => el.type, allInputs[i]);
        const name = await page.evaluate(el => el.name || '', allInputs[i]);
        const placeholder = await page.evaluate(el => el.placeholder || '', allInputs[i]);
        log(`     [${i}] type=${type}, name=${name}, placeholder=${placeholder}`);
      }
      
      await takeScreenshot(page, '02-port-field-not-found');
      results.tests.push({ name: 'Port Field Exists', status: 'failed', error: 'Port input not found' });
      results.errors.push({ test: 'Port Field Exists', error: 'Port input not found' });
      return { exists: false };
    }
  } catch (error) {
    log(`   ❌ Error: ${error.message}`);
    results.tests.push({ name: 'Port Field Exists', status: 'failed', error: error.message });
    results.errors.push({ test: 'Port Field Exists', error: error.message });
    return { exists: false };
  }
}

async function testChangePort(page, portInput) {
  log('📍 Test 3: Change Port Value');
  try {
    // Clear the field and enter new value
    await portInput.tripleClick();
    await portInput.type('5000');
    await delay(500);
    
    const newValue = await page.evaluate(el => el.value, portInput);
    log(`   ✓ Port changed to: ${newValue}`);
    await takeScreenshot(page, '03-port-changed');
    
    results.tests.push({ name: 'Change Port', status: 'passed', newValue });
    return true;
  } catch (error) {
    log(`   ❌ Error: ${error.message}`);
    results.tests.push({ name: 'Change Port', status: 'failed', error: error.message });
    results.errors.push({ test: 'Change Port', error: error.message });
    return false;
  }
}

async function testSaveButton(page) {
  log('📍 Test 4: Find Save Button');
  try {
    // Look for save button - could be text "Save" or similar
    const saveButton = await page.$('button:has-text("Save")') || 
                       await page.$('button:has-text("Save Settings")') ||
                       await page.$('button[type="submit"]');
    
    if (saveButton) {
      const buttonText = await page.evaluate(el => el.textContent.trim(), saveButton);
      log(`   ✓ Save button found: "${buttonText}"`);
      await takeScreenshot(page, '04-save-button-found');
      results.tests.push({ name: 'Save Button Exists', status: 'passed', buttonText });
      return true;
    } else {
      log('   ⚠ Save button not found');
      await takeScreenshot(page, '04-save-button-not-found');
      results.tests.push({ name: 'Save Button Exists', status: 'failed', error: 'Save button not found' });
      results.errors.push({ test: 'Save Button Exists', error: 'Save button not found' });
      return false;
    }
  } catch (error) {
    log(`   ❌ Error: ${error.message}`);
    results.tests.push({ name: 'Save Button Exists', status: 'failed', error: error.message });
    results.errors.push({ test: 'Save Button Exists', error: error.message });
    return false;
  }
}

async function generateReport() {
  log('');
  log('========================================');
  log('TEST REPORT');
  log('========================================');
  log('');
  
  const passed = results.tests.filter(t => t.status === 'passed').length;
  const failed = results.tests.filter(t => t.status === 'failed').length;
  
  log(`Total Tests: ${results.tests.length}`);
  log(`Passed: ${passed}`);
  log(`Failed: ${failed}`);
  log('');
  
  log('Test Details:');
  results.tests.forEach((test, index) => {
    const status = test.status === 'passed' ? '✓' : '✗';
    log(`  ${status} ${index + 1}. ${test.name}`);
    if (test.value) log(`     Value: ${test.value}`);
    if (test.newValue) log(`     New Value: ${test.newValue}`);
    if (test.buttonText) log(`     Button: ${test.buttonText}`);
    if (test.error) log(`     Error: ${test.error}`);
  });
  
  if (results.screenshots.length > 0) {
    log('');
    log('Screenshots:');
    results.screenshots.forEach(screenshot => {
      log(`  - ${screenshot}`);
    });
  }
  
  log('');
  log('========================================');
  
  return failed === 0;
}

// Main test function
async function runTests() {
  log('🚀 Starting OpenCode Settings Tests');
  log(`   URL: ${BASE_URL}${SETTINGS_PATH}`);
  log('');

  const browser = await puppeteer.launch({
    headless: process.env.HEADLESS !== 'false',
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--disable-dev-shm-usage']
  });

  try {
    const page = await browser.newPage();
    await page.setViewport({ width: 1280, height: 800 });
    
    // Run tests
    await testPageLoad(page);
    const portResult = await testPortFieldExists(page);
    
    if (portResult.exists && portResult.element) {
      await testChangePort(page, portResult.element);
      await testSaveButton(page);
    }
    
    // Generate report
    const success = await generateReport();
    
    return success;
    
  } catch (error) {
    log('');
    log('❌ Fatal error during testing:');
    log(`   ${error.message}`);
    log(`   ${error.stack}`);
    return false;
  } finally {
    await browser.close();
    log('');
    log('🛑 Browser closed');
  }
}

// Run the tests
if (require.main === module) {
  runTests()
    .then(success => {
      process.exit(success ? 0 : 1);
    })
    .catch(error => {
      console.error('Unhandled error:', error);
      process.exit(1);
    });
}

module.exports = { runTests };
