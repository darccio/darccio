var vCard = require('vcards-js');

vc = vCard();
vc.firstName = 'Dario';
vc.lastName = 'Castañé';
vc.birthday = new Date('23-09-1985');
vc.title = 'Cloud Architect & Full Stack Developer';
vc.url = 'https://da.rio.hn';
vc.gender = 'M';
vc.cellPhone = '+34665269218';
vc.email = 'd@rio.hn';

vc.workPhone = '+34935511443';
vc.workUrl = 'https://loyal.guru';
vc.workEmail = 'dcastane@loyal.guru';
vc.workAddress.street = 'Carrer de Pallars 85, local 4 baixos';
vc.workAddress.city = 'Barcelona';
vc.workAddress.postalCode = '08018';
vc.workAddress.stateProvince = 'Barcelona';
vc.workAddress.countryRegion = 'Catalonia';

vc.socialUrls['github'] = 'https://github.com/imdario';
vc.socialUrls['linkedin'] = 'https://www.linkedin.com/in/darccio';
vc.socialUrls['telegram'] = 'https://t.me/darccio';
vc.socialUrls['twitter'] = 'https://twitter.com/im_dario';

vc.logo.embedFromFile('./static/images/default.png');
vc.photo.embedFromFile('./static/images/logo.png');

vc.source = 'https://da.rio.hn/.well-known/carddav';

vc.saveToFile('./static/dario-castane.vcf');
