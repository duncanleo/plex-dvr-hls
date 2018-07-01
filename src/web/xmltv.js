const Mustache = require('mustache');
const fs = require('fs');
const dateFormat = require('dateformat');

const channels = require('../../channels.json');

const xmltv = (req, res) => {
  res.send(
    Mustache.render(
      fs.readFileSync('templates/xmltv.mustache', { encoding: 'utf8' }),
      {
        channels: channels.map((c, index) => ({
          id: index + 1,
          name: c.name,
          today: dateFormat(new Date(), 'yyyymmdd'),
        })),
      },
    ),
  );
};

module.exports = {
  xmltv,
};
