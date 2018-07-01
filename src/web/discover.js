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
        BaseURL: `http://${req.headers.host}`,
        LineupURL: `http://${req.headers.host}/lineup.json`,
        Manufacturer: 'Silicondust',
      },
    ),
  );
};

module.exports = {
  discover,
};
