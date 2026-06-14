'use strict';

(function () {
  var form = document.getElementById('kc-totp-settings-form') ||
             document.getElementById('kc-otp-login-form');

  if (!form) return;

  var codeInput = form.querySelector('input[name="totp"], input[name="otp"]');
  if (!codeInput) return;

  codeInput.setAttribute('inputmode', 'numeric');
  codeInput.setAttribute('pattern', '[0-9]*');
  codeInput.setAttribute('autocomplete', 'one-time-code');
  codeInput.setAttribute('maxlength', '6');

  codeInput.addEventListener('input', function (e) {
    e.target.value = e.target.value.replace(/\D/g, '').slice(0, 6);
  });

  codeInput.addEventListener('paste', function (e) {
    var pasted = (e.clipboardData || window.clipboardData).getData('text');
    var digits = pasted.replace(/\D/g, '').slice(0, 6);
    e.preventDefault();
    e.target.value = digits;
    if (digits.length === 6) {
      form.submit();
    }
  });

  // Auto-submit when 6 digits are entered
  codeInput.addEventListener('input', function () {
    if (codeInput.value.length === 6) {
      form.submit();
    }
  });

  // Hotel Key branding: show countdown timer for TOTP validity window
  var timerEl = document.getElementById('kc-totp-timer');
  if (timerEl) {
    function updateTimer() {
      var now = Math.floor(Date.now() / 1000);
      var remaining = 30 - (now % 30);
      timerEl.textContent = remaining + 's';
      timerEl.style.color = remaining <= 5 ? '#d32f2f' : '#388e3c';
    }
    updateTimer();
    setInterval(updateTimer, 1000);
  }
})();
