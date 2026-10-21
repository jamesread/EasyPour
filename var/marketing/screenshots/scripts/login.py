"""Logs in as the fictional admin "alex" when the login form is showing."""
import time

from selenium.webdriver.common.by import By

LOGIN_JS = """
const done = arguments[0];
fetch('/login', {
  method: 'POST',
  headers: {'Content-Type': 'application/json'},
  body: JSON.stringify({username: 'alex', password: 'demo-password'}),
}).then(r => done(r.status)).catch(e => done(String(e)));
"""


def run(driver):
    if not driver.find_elements(By.CSS_SELECTOR, ".login-view"):
        return
    status = driver.execute_async_script(LOGIN_JS)
    if status != 200:
        raise RuntimeError("demo login failed: %s" % status)
    driver.refresh()
    time.sleep(3)
