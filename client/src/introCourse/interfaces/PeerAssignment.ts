export interface PeerInfo {
  courseParticipationID: string
  gitlabUsername: string
  seatName: string
  tutorGitlabUsername: string
}

export interface OwnPeerAssignment {
  peersIReview: PeerInfo[]
  peersWhoReviewMe: PeerInfo[]
}

export interface PeerAssignment {
  studentID: string
  peerID: string
}

export interface SyncResult {
  studentID: string
  peerID: string
  success: boolean
  error?: string
}
