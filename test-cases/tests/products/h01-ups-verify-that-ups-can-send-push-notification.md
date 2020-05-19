---
---

# H01 - UPS: Verify that UPS can send push notification

## Prerequisites

- real Android device with latest OS version
- real iOS device with latest OS version
- Apple Developer Program Account (this is not normal Apple account, but paid one)
- Apple Mac

## Steps

### Android preparation

1. `git clone https://github.com/aerogear/unifiedpush-cookbook.git`
2. `cd unifiedpush-cookbook/cordova/HelloWorld`
3. Follow instructions in the [README](https://github.com/aerogear/unifiedpush-cookbook/tree/master/cordova/HelloWorld)

### iOS

1. `git clone https://github.com/aerogear/unifiedpush-cookbook.git`
2. `cd unifiedpush-cookbook/cordova/HelloWorld`
3. Replace the bundleId with your bundleId (the one associated with your certificate), by editing the config.xml at the root of this project, change the id attribute of the widget node.
4. `cordova platform add ios`
5. `open platforms/ios/AeroGear\ UnifiedPush\ HelloWorld.xcworkspace`
6. In signing settings select your your apple developer account under `Team`
7. Follow the [official Apple guide](https://help.apple.com/xcode/mac/current/#/devdfd3d04a1) to enable push notifications for your Xcode project.
8. Follow the [official Apple guide](https://help.apple.com/developer-account/#/dev82a71386a) to generate an APNs client TLS certificate and export the client TLS identity from your Mac. Make sure to protect the p12 file with a password. The exported p12 file with the password will be used later when binding your Mobile App to the Push Notifications.
9. Create application and iOS variant in UPS with your exported p12 file.
10. Update `src/push-config.js` with correct variant ID and secret.
11. Close Xcode
12. `cordova platform remove ios`
13. `cordova platform add ios`
14. `open platforms/ios/AeroGear\ UnifiedPush\ HelloWorld.xcworkspace`
15. Make sure `Team` is set
16. Run the app
17. Send notification from UPS
    > You should see the notification on your device
