# Use Cases

## UC1 - Provide boot image

Actor:
1. Server in PXE mode on initial bootup
1. Server agent on consecutive bootups

Trigger:
`/v1/boot/{mac}`

API contract with Pixiecore

Brief:
Server requests boot image

Basic flow:
1. Request MAAS API for the image
1. Decide response:
    1. Success: Prepare appropriate response
    1. Failure: Respond with given error

Note:
From the perspective of PXE API the functional purpose of the request (discovery or OS) has no impact on implementation.

## UC2 - Report server update

Actor:
Server agent

Trigger:
`/v1/update/{mac}/{state}`

States:
1. DISCOVERED
1. IMAGE_BURNED
1. IMAGE_BURN_FAILED
1. SERVER_RUNNING
1. ...

Brief:
Register server updates.

Basic flow:
1. Inform MAAS API about server change
1. Respond with given response

Note:
From the perspective of PXE API the functional purpose of the update request (discovery details, lifecycle change, etc.) has no impact on implementation.
