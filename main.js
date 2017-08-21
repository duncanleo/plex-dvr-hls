const express = require('express');
const morgan = require('morgan');

const { lineup, lineupStatus, discover, capability, stream } = require('./src/web');

const port = process.env.PORT || 5004;
const app = express();

app.use(morgan('dev'));

app.get('/lineup.json', lineup);
app.get('/lineup_status.json', lineupStatus);
app.get('/discover.json', discover);
app.get('/capability', capability);
app.get('/stream/:channelNum', stream);

app.listen(port, () => {
  console.log(`Started listening on port ${port}`);
});
