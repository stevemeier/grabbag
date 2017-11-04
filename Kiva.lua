-- 
-- MoneyMoney App Extension for Kiva.org
-- https://moneymoney-app.com
-- https://www.kiva.org
--
-- Author: Steve Meier (email ät steve minus meier punkt de)
-- Homepage: https://www.steve-meier.de
--
-- History
-- 20170603: First working version
-- 20170607: Fix SupportsBank function to return only Kiva.org

WebBanking {
  version = 20170607,
  services = {"Kiva.org"},
  description = "Kiva.org Module",
  url = "https://www.kiva.org/login"
}

local connection
local overview_html

function SupportsBank (protocol, bankCode)
  return protocol == ProtocolWebBanking and bankCode == "Kiva.org"
end

function InitializeSession (protocol, bankCode, username, username2, password, username3)
  MM.printStatus("Logging in...")

  -- Create HTTPS connection object
  connection = Connection()
  connection.language = "de-de"

  -- Fetch login page
  local loginPage = HTML(connection:get(url))

  -- Fill in login credentials
  loginPage:xpath("//*[@id='loginForm_email']"):attr("value", username)
  loginPage:xpath("//*[@id='loginForm_pass']"):attr("value", password)

  -- Submit login form.
  local request = connection:request(loginPage:xpath("//*[@id='loginForm_submit']"):click())

  overview_html = HTML(request)
  -- After a failed login the body has an id "login"
  -- If login is successful the body has an id "portfolio"
  local login = overview_html:xpath("//*[@id='login']")
  if login:length() > 0 then
    MM.printStatus("Login failed!");
    return LoginFailed
  else
    MM.printStatus("Login successful");
    return nil
  end
end

function ListAccounts (knownAccounts)
  -- Return array of accounts.
  local account = {
    name = "Kiva.org",
    accountNumber = "1",
    currency = "USD"
  }

  return {account}
end

function RefreshAccount (account, since)
  -- Get outstanding and available credit from HTML
  outstanding = overview_html:xpath("(//span[@class='statistic kiva-green'])[1]"):text()
  available   = overview_html:xpath("(//span[@class='statistic kiva-green'])[2]"):text()

  print("Outstanding: " .. outstanding .. " | Available: " .. available)
  print("Removing formatting from numbers")

  -- Remove the dollar sign
  outstanding = outstanding:gsub("%$", '')
  available = available:gsub("%$", '')
  -- Remove the comma (if present)
  outstanding = outstanding:gsub(",", '')
  available = available:gsub(",", '')

  local total = outstanding + available
  print("Outstanding: " .. outstanding .. " | Available: " .. available .. " | Total: " .. total)
  MM.printStatus("Total: USD " .. total);

  -- Return the balance, but no transactions
  return {balance = total, transactions = {}}
end

function EndSession ()
  -- Logout
  local logout = HTML(connection:get('https://www.kiva.org/logout'))
  
  return nil
end

-- SIGNATURE: MCwCFHw2VHMmaqsVzY4jrrUYIrmCjcxhAhRDk+sZUeli4tAsKG/fTQff7SN4qg==
