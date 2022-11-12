(function () {

    class DeviceList {
        constructor() {
            this.element = document.getElementById('devices');
            this.devices = [];
        }

        update(devices) {
            this.element.classList.remove('loading');
            this.devices = [];
            this.element.innerHTML = '';

            devices.forEach((info) => {
                const device = new Device(info);
                this.element.appendChild(device.element);
                this.devices.push(device);
            });
        }

        showError(err) {
            this.element.innerHTML = '';
            const errorElem = makeElem('div', 'devices-error', { textContent: err });
            this.element.appendChild(errorElem);
        }
    }

    class Device {
        constructor(info) {
            this.info = info;
            this.status = null;

            this.name = makeElem('label', 'device-name', { textContent: info.name });
            this.onOff = makeElem('div', 'device-on-off');
            this.onOff.addEventListener('click', () => this.toggleOnOff());

            this.brightnessButton = makeElem(
                'button',
                'brightness-button device-color-controls-button',
            );
            this.brightnessButton.addEventListener('click', () => this.editBrightness());
            this.colorButtonSwatch = makeElem('div', 'color-button-swatch');
            this.colorButton = makeElem(
                'button',
                'color-button device-color-controls-button',
                {},
                [this.colorButtonSwatch],
            );
            this.colorButton.addEventListener('click', () => this.editColor());
            this.colorControls = makeElem('div', 'device-color-controls', {}, [
                this.brightnessButton, this.colorButton,
            ]);

            this.error = makeElem('label', 'device-error');
            this.error.style.display = 'none';
            this.loader = makeElem('div', 'loader');

            this.element = makeElem('div', 'device', {}, [
                this.name, this.onOff, this.colorControls, this.error, this.loader,
            ]);

            if (info['status']['is_online']) {
                this.updateStatus(info['status']);
            } else {
                // Try again, get an error message.
                this.fetchUpdate();
            }
        }

        updateStatus(status) {
            this.status = status;
            this.element.classList.remove('device-loading');
            this.element.classList.remove('loading');
            this.error.style.display = 'none';
            if (status === null) {
                this.element.classList.add('device-offline');
            } else {
                this.element.classList.remove('device-offline');
                if (status["is_on"]) {
                    this.onOff.classList.add('device-on-off-on');
                } else {
                    this.onOff.classList.remove('device-on-off-on');
                }
                this.brightnessButton.textContent = status["brightness"] + "%";
                this.colorButtonSwatch.style.backgroundColor = previewColor(status);
            }
        }

        showError(err) {
            this.updateStatus(null);
            this.error.textContent = err;
            this.error.style.display = 'block';
        }

        fetchUpdate() {
            this.doCall(lightAPI.getStatus(this.info.id));
        }

        toggleOnOff() {
            const newOn = !this.status['is_on'];
            this.doCallChecked(
                lightAPI.setOnOff(this.info.id, newOn),
                (status) => status['is_on'] == newOn,
            );
        }

        editBrightness() {
            const popup = new window.controlPopups.BrightnessPopup(this.status['brightness']);
            popup.onBrightness = (value) => {
                this.doCallChecked(
                    lightAPI.setBrightness(this.info.id, value),
                    (status) => status['brightness'] == value,
                );
            };
            popup.open();
        }

        editColor() {
            const popup = new window.controlPopups.ColorPopup(this.status);
            popup.onRGB = (rgb) => {
                this.doCallChecked(
                    lightAPI.setRGB(this.info.id, rgb),
                    (status) => (status['rgb'][0] == rgb[0] && status['rgb'][1] == rgb[1] &&
                        status['rgb'][2] == rgb[2]),
                );
            };
            popup.onTone = (tone) => {
                this.doCallChecked(
                    lightAPI.setTone(this.info.id, tone),
                    (status) => status['color_tone'] == tone,
                );
            };
            popup.open();
        }

        doCall(promise) {
            this.element.classList.add('device-loading');
            this.element.classList.add('loading');
            promise.then((status) => {
                this.updateStatus(status);
            }).catch((err) => {
                this.showError(err);
            });
        }

        doCallChecked(promise, check) {
            this.element.classList.add('device-loading');
            this.element.classList.add('loading');
            promise.then((status) => {
                if (!check(status)) {
                    // The status change might have been delayed,
                    // so we will attempt to fetch it again.
                    setTimeout(() => this.fetchUpdate(), 1000);
                } else {
                    this.updateStatus(status);
                }
            }).catch((err) => {
                this.showError(err);
            });
        }
    }

    function previewColor(status) {
        if (status['use_rgb']) {
            return rgbToHex(status['rgb']);
        } else {
            return toneColor(status['color_tone']);
        }
    }

    window.addEventListener('load', () => {
        window.deviceList = new DeviceList();
        lightAPI.getDevices().then((devs) => {
            window.deviceList.update(devs);
        }).catch((err) => {
            window.deviceList.showError(err);
        })
    });

})();
