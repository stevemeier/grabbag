#!/usr/bin/perl

# Quick PoC to store and receives files via DNS
# Date: December 29th, 2017
# Author: Steve Meier

# Load all required modules
use strict;
use warnings;
use Carp;
use Digest::SHA;
use File::Basename qw(basename);
use Getopt::Long;
use MIME::Base64 qw(decode_base64 encode_base64);
use Net::DNS;

my ($encode, $download, $dnsname, $output, $ttl);
GetOptions("encode=s"   => \$encode,
	   "download"   => \$download,
           "dnsname=s"  => \$dnsname,
           "output=s"   => \$output,
           "ttl=s"      => \$ttl);
	 
# Encode a file into Base64 and DNS TXT records   
if ($encode) {
  my $base64data;
  my $txtdata;
  my $i = 0;
  my $hash;
  my $size;
  my $name;

  # Default TTL. Can be overridden with --ttl
  if (not(defined($ttl))) { $ttl = 86400 };

  if (not(-f $encode)) {
    print STDERR "ERROR: Please specify a file to encode!\n";
    exit 1;
  }

  if (not(defined($dnsname))) {
    print STDERR "ERROR: Please specify DNS name!\n";
    exit 2;
  }

  # Ensure we use a fully qualified name, ending with dot
  if (not($dnsname =~ /\.$/x)) { $dnsname .= "." };

  # We store meta information such as original file name, size and checksum
  $name = basename($encode);
  $size = filesize($encode);
  $hash = filehash($encode);

  # Read the file in one go
  open(my $FILE, "<", $encode) || croak "ERROR: Could not read file!\n";
  local($/) = undef;
  $base64data = encode_base64(<$FILE>, '');  
  close($FILE) || croak "ERROR: Could not close file!\n";

  # Print the base record with meta information
  print "$dnsname $ttl IN TXT \"$name\" \"$size\" \"sha256\" \"$hash\"\n";

  # Print the data records
  while (length($base64data) > 0) {
    $txtdata = substr $base64data, 0, 255, '';
    print i_to_label($i).".$dnsname $ttl IN TXT \"$txtdata\"\n";
    $i++;
  }

  exit 0;
}

if ($download) {
  my ($reply, $rr);
  my @filemeta;
  my $base64data;
  my $i = 0;
  my $filename;

  # Create a resolver object
  my $res = Net::DNS::Resolver->new;

  # Look up the files meta information (name, size, hashalg, hash)
  $reply = $res->query($dnsname, 'TXT');
  if ($reply) {
    foreach my $rr (grep { $_->type eq 'TXT' } $reply->answer) {
      @filemeta = $rr->txtdata;
    }
  }

  # Do lookups until we encounter "nxdomain" which indrectly marks end of file
  until ($res->errorstring =~ /nxdomain/i) {
    $reply = $res->query(i_to_label($i).".$dnsname", 'TXT');
    if ($reply) {
      foreach my $rr (grep { $_->type eq 'TXT' } $reply->answer) {
        $base64data .= $rr->txtdata;
      }
    }
    $i++;
  }

  # Save to original or provided filename
  if ($output) {
    $filename = $output;
  } else {
    $filename = $filemeta[0];
  }

  # Write base64 decoded file to disk
  open(my $FILE, ">", $filename) || croak "ERROR: Could not open $filename for writing";
  print $FILE decode_base64($base64data);
  close($FILE) || croak "ERROR: Could not close $filename";
  
  # Check file size is correct
  if (filesize($filename) != $filemeta[1]) {
    print STDERR "ERROR: File size does not match!\n";
    exit 1;
  } 

  # Check file hash is correct
  if (filehash($filename) ne $filemeta[3]) {
    print STDERR "ERROR: File hash does not match!\n";
    exit 2;
  }

  print STDERR "INFO: File downloaded successfully to $filename\n";
  exit 0;
}

print "---\n";
print "Encode a file to zone file format:\n";
print "$0 --encode <FILE> --dnsname <somename.foobar.com>\n\n";
print "Download a file from DNS:\n";
print "$0 --download --dnsname <somename.foobar.com> [ --output <FILE> ]\n";
exit;

sub i_to_label {
  my ($i) = @_;
  my $result = '';

  while ($i >= 26) {
    $result .= 'z';
    $i -= 26;
  }

  $result .= chr(97 + $i);

  return $result;
}

sub filehash {
  my ($filename, $hashalg) = @_;
  if (not(defined($hashalg))) { $hashalg = "sha256" };

  my $sha = Digest::SHA->new($hashalg);
  $sha->addfile($filename);

  return $sha->hexdigest;
}

sub filesize {
  my ($filename) = @_;

  my $size = (stat $filename)[7];

  return $size;
}
