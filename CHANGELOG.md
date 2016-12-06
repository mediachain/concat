## 6/12/2016: concat-v1.3.1
- Bug fix: crash when setting auth rules [[Issue #93]](https://github.com/mediachain/concat/issues/93)

## 5/12/2016: concat-v1.3
- Datastore garbage collection [[PR #88]](https://github.com/mediachain/concat/pull/88)
- IPFS DHT integration [[PR #90]](https://github.com/mediachain/concat/pull/90)

## 21/11/2016: concat-v1.2
- Push publishing for authorized peers [[PR #82]](https://github.com/mediachain/concat/pull/82)
- More robust public IP address detection [[PR #84]](https://github.com/mediachain/concat/pull/84)
- Multiaddr compatible peer handles [[PR #79]](https://github.com/mediachain/concat/pull/79)
- Improved peer lookup logic with libp2p-reported connectedness [[PR #75]](https://github.com/mediachain/concat/pull/75)

## 4/11/2016: concat-v1.1
- New directory: `/ip4/52.7.126.237/tcp/9000/QmSdJVceFki4rDbcSrW7JTJZgU9so25Ko7oKHE97mGmkU6`
- Add a deps field to statement body in order to support object dependency merging [[Issue #63]](https://github.com/mediachain/concat/issues/63)
- Optimize data fetching in the merge protocol [[Issue #44]](https://github.com/mediachain/concat/issues/44)
- Support compound statement publication [[Issue #62]](https://github.com/mediachain/concat/issues/62)
- Implement user write locking in statement db, allow concurrent reads with long running queries [[PR #73]](https://github.com/mediachain/concat/pull/73)
- Add delay in the offline->public transition when NAT config is auto, allow port mapping to complete before directory registration [[PR #72]](https://github.com/mediachain/concat/pull/72)
- Automate binary builds for releases [[Issue 56]](https://github.com/mediachain/concat/issues/56)

## 24/10/2016: concat-v1.0
- First release; baseline feature completeness.
