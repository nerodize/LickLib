package de.licklib.kotlinbackend.model

import de.licklib.kotlinbackend.entity.FileEntity
import java.io.InputStream
import java.util.UUID

data class File(
    override val id: UUID? = null,
    override val name: String,
    override val contentType: String,
    override val duration: Long,
    override val size: Long,
    val inputStream: InputStream,
) : FileEntity(
    id = id,
    name = name,
    contentType = contentType,
    duration = duration,
    size = size,
)