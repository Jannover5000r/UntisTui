# Terminal app for Webuntis timetable system. WIP

## Set up by cloning the repo and create a .env file with the Following credential 
- UNTIS_USERNAME
- UNTIS_PASSWORD
- UNTIS_URL
  where UNTIS_URL will be following the structure "https://thalia.webuntis.com/WebUntis/jsonrpc.do?school= [your School]"
  your school name can be found by navigating to the login screen of your school's untis webpage following this struct:
  "https://mtg.webuntis.com/WebUntis/?school= [your School]#/basic/login"
  Than build the program and run the binary in the same folder as the .env and it will create folders containing the timetable in this same directory and will start the TUI with your Timetable
