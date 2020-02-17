# followup - Simple email reminders

### What is this?

followup allows you to quickly schedule email reminders by forwarding important emails to a specially constructed address. This can help you keep track of tasks, deadlines, etc.

### How does it work?

The solution consists of two pieces of software:

- **followup**

  Reads incoming emails from Stdin and parses the recipient addresses to determine when you want to be reminded. It then stores this information along with the message subject, id, and timestamp in a SQLite database.

- **followup-daemon**

  Monitors the SQLite database for reminders which are due and sends them out using an SMTP gateway, marking them as sent.

- **check_followup_daemon.sh**

  This Nagios-style plugin monitors pending reminders to alert you if there is a problem sending them.

### Supported reminder formats

You can specify the date and time of your reminders in different formats.
Here are some example:

- 3h -- Three hours from now
- 2d -- Two days from today (same as 48h)
- 1w -- One week from day (same as 7d)
- 1m -- One month from today
- 2y -- Two years from today
- 8pm -- 8 o'clock in the afternoon
- 2000 - 8 o'clock (using 24h scheme)
- monday - Next monday
- nov13 - November 13th (month first)
- 13nov - November 13th (day first)



 