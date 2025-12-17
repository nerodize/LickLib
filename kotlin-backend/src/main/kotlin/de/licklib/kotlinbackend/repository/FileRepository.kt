package de.licklib.kotlinbackend.repository

import de.licklib.kotlinbackend.entity.FileEntity
import org.springframework.data.jpa.repository.JpaRepository
import java.util.UUID

interface FileRepository : JpaRepository<FileEntity, UUID>