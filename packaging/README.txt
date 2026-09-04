3D Job Desk Printer Bridge
==========================

This program runs on a computer that can ping your printers. It only makes
outbound HTTPS calls to https://app.3djobdesk.com. It does not open a port
on your network.

Windows
-------
Download the installer from Printers and run it. It installs a real app with
the 3D Job Desk icon, not a terminal window.

1. Leave "Start with Windows" checked if this shop PC should connect at sign-in.
2. On the Printers page, click "Create pairing code".
3. Paste the code and click Connect. The website field is already filled in.
4. Closing the window keeps the bridge in the system tray.
5. Right-click the tray icon → Quit to stop. You can change "Start with Windows"
   later in the app.

Mac / Linux
-----------
Download the single program for this computer. chmod +x if needed, then run it.
Enter the pairing code from Printers. The website is always
https://app.3djobdesk.com.

Security
--------
- Pairing codes expire in 15 minutes and can be used once.
- This computer stores a secret that only your desk can revoke.
- The program will not contact public IPs or cloud metadata addresses.
- It only asks Moonraker for printer status. It cannot send gcode.
- Disconnect a computer anytime from Printers.

Requirements
------------
Windows 10+, macOS 12+, or a 64-bit Linux PC.
Raspberry Pi: use the Linux ARM build on 64-bit Raspberry Pi OS, or a shop PC
on the same LAN.
