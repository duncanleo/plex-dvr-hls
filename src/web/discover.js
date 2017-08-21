const config = require('../config');

const discover = (req, res) => {
  res.setHeader('Content-Type', 'application/json');
  res.send(
    JSON.stringify(
      {
        FriendlyName: config.name,
        ModelNumber: 'HDTC-2US',
        FirmwareName: 'hdhomeruntc_atsc',
        TunerCount: 1,
        FirmwareVersion: '20150826',
        DeviceID: (Math.floor(Math.random() * 90000000) + 10000000).toString(),
        DeviceAuth: 'test1234',
        BaseURL: 'http://192.168.1.79:5004',
        LineupURL: 'http://192.168.1.79:5004/lineup.json',
        Manufacturer: 'Silicondust',
      },
    ),
  );
};

module.exports = {
  discover,
};
