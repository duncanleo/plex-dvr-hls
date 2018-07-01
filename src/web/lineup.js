const channels = require('../../channels.json');

const lineup = (req, res) => {
  res.setHeader('Content-Type', 'application/json');
  res.send(
    JSON.stringify(
      channels.map((channel, index) => ({
        GuideNumber: (index + 1).toString(),
        GuideName: channel.name,
        Tags: [],
        URL: `http://${req.headers.host}/stream/${index + 1}`,
      })),
    ),
  );
};

const lineupStatus = (req, res) => {
  res.setHeader('Content-Type', 'application/json');
  res.send(
    JSON.stringify(
      {
        ScanInProgress: 0,
        ScanPossible: 1,
        Source: 'Cable',
        SourceList: ['Cable'],
      },
    ),
  );
};

module.exports = {
  lineup,
  lineupStatus,
};
