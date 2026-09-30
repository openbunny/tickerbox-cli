Output demo/tickerbox.gif

Require tickerbox

Set Shell "bash"
Set FontSize 18
Set Width 1200
Set Height 760
Set Padding 20
Set TypingSpeed 40ms

Type "tickerbox status"
Enter
Sleep 3s

Type "tickerbox doctor"
Enter
Sleep 2500ms

Type "tickerbox ticker list"
Enter
Sleep 1500ms

Type "tickerbox ticker template list"
Enter
Sleep 1500ms

Type "tickerbox --json status"
Enter
Sleep 3s

Type "tickerbox tz list --grep Europe"
Enter
Sleep 1500ms

Type "tickerbox version"
Enter
Sleep 2s
