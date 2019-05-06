Installation and usage
======================
PLEASE REMEMBER THIS IS INCOMPLETE SOFTWARE, AND 
MAY HAVE SECURITY ISSUES AND BUGS. USE AT YOUR OWN
RISK. 

At the end of the guide, you will have an email
server that lets you send and receive emails 
from your own domain.

Before you begin
----------------
You will need a domain name, e.g. `mfashby.net` or 
`janedoe.com`. I use [Namecheap](https://www.namecheap.com/)
, but other domain name providers are available.
If you want to follow this guide exactly, you will 
need to create an account with Namecheap and log in.

You will need a server with a fixed IP address. For
this guide I am going to use a server from 
[Digital Ocean](https://www.digitalocean.com/). Other 
providers are available. The software will run on 
most operating systems. If you want to follow this 
guide exactly, you will need to create an account 
with Digital Ocean and log in.

Set up a server
---------------
Log into Digital Ocean. Select New Project and fill the required fields.

![digital ocean new project screen](img/digitalocean_createnew.png) 

Create a new droplet. I'm going to create one with the Ubuntu 16.04
operating system. Henrymail doesn't need much resource, so I'm going
to use the cheapest price plan.

![digital ocean new droplet screen](img/digitalocean_newdroplet.png)

Once you have created your droplet, write down it's IP address. 
This should be visible on the Project page. 

![digital ocean project page with droplet](img/digitalocean_droplet_overview.png)

You will be emailed the username and password for accessing the droplet. 
Click on `...` > `Access Console` to log into the droplet, and enter the 
username and password that were emailed to you. You will be prompted to 
enter a new password. Don't forget it! Once you have entered the new 
password, you'll be presented with a terminal like this in your browser:

![digital ocean droplet console](img/digitalocean_droplet_terminal.png)  

You can then copy & paste the following command into the terminal and 
press enter to run it. This will download and install the software.
You will be prompted for the domain you want to serve (e.g. mfashby.net)

The administrator username and password will be shown in the terminal. 
Note them down. Once the script has finished, the server will be running!
 
```bash
wget https://raw.githubusercontent.com/MFAshby/henrymail/master/install.sh \
 && chmod +x install.sh \
 && sudo ./install.sh
```

Next we need to configure the DNS for your server. Log into Namecheap
and navigate to `Dashboard` > `Manage` > `Advanced DNS`. 

Add a new A record for host `mail`, and fill in the IP address of your digital 
ocean droplet as the value (you wrote this down earlier). Click the tick to 
save it.
![namecheap add A record](img/namecheap_add_a_record.png) 

Allow a minute or so for this record to get updated across the DNS system. 
You can check it's progress on [whatsmydns.net](https://www.whatsmydns.net)

Open your web browser to access the administration 
interface, e.g. https://mail.mfashby.net/, and enter the username and password
. Save the password to your password manager, or you can change the password 
to something more memorable here if you want.


Now you can enable it as a service so it will start whenever your droplet 
is restarted. Press `ctrl+c` to stop henrymail, then run the following:
```bash
systemctl daemon-reload
systemctl enable henrymail
systemctl start henrymail
```

Refresh the administration page again to check everything is working OK. 
You can now close the terminal!

We've almost got a working email server. Go back to Namecheap, and add 
a Custom MX record with host set to `@`, and value set to your mail server's name (with a trailing dot!):
![namecheap add mx record screenshot](img/namecheap_add_mx.png)

Finally, you can connect your email client. Here I am using [Thunderbird](https://www.thunderbird.net)

Add a new account, and enter the details as follows:
* Incoming server should be IMAP, with your mail server name e.g. `mail.mfashby.net`
* Outgoing server should be SMTP, with your mail server name e.g. `mail.mfashby.net`
* Press 're-test', and thunderbird should automatically pick up the 
port numbers and SSL settings.
![thunderbird new account sample](img/thunderbird_account_config.png)  

Press Done, and enter the admin password when prompted

You're done! Test your new configuration by sending an email to another
address that you own (or a friend)

Keep reading for further improvements to your server

SPF and DKIM records
====================
It's advisable to have Sender Policy Framework and DomainKeys Identified Mail 
records set up for your server. These help other servers to validate emails 
from your server, so your emails are less likely to be rejected. See 
[this article](https://blog.woodpecker.co/cold-email/spf-dkim/) for an explanation.

To set these up, log into the administration page, then click 'Health Checks'.
![henrymail health checks page](img/henrymail_healthchecks.png)

These text boxes show the entries you need to add to your DNS system in order 
for emails to be verified by other servers. Log into Namecheap and add the 
corresponding entries as TXT records:

![namecheap adding SPF and DKIM records](img/namecheap_spf_dkim.png)

Save these entries, and allow a couple of minutes for them to propagate through
the DNS system. Reload the 'Health Checks' page, it should now display a message
saying that your SPF and DKIM records are OK

![henrymail health checks OK](img/henrymail_healthchecks_ok.png)


Advanced Configuration
======================
Henrymail is compatible with some more advanced setups, e.g. if you are running a 
web server on the same server.

The complete list of configuration options is documented in 
[henrymail.full.prop](../henrymail.full.prop)

TODO
====
* Administration page guide to adding users.
* Ongoing administration (quotas, spam settings)
